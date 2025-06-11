# Firecracker Process Management Flows

## Process State Machine

```mermaid
stateDiagram-v2
    [*] --> Starting : createNewProcess()
    Starting --> Ready : socket available & ping success
    Starting --> Error : startup timeout or failure
    Ready --> Busy : VM assigned (GetOrCreateProcess)
    Busy --> Ready : VM released (process reuse model)
    Busy --> Stopping : VM deleted (1:1 model)
    Ready --> Stopping : manager shutdown
    Stopping --> Stopped : SIGTERM success
    Stopping --> Stopped : SIGKILL after timeout
    Error --> Stopped : cleanup
    Stopped --> [*]
    
    note right of Ready : Socket available\nHTTP API responsive
    note right of Busy : VM configured and running\nMetrics streaming active
    note right of Stopping : Graceful shutdown (5s)\nthen force kill
```

## VM Lifecycle State Transitions

```mermaid
stateDiagram-v2
    [*] --> Creating : CreateVM API call
    Creating --> Created : Firecracker configured
    Creating --> Failed : Configuration error
    Created --> Booting : BootVM API call
    Booting --> Running : InstanceStart success
    Booting --> Failed : Boot failure
    Running --> Paused : PauseVM API call
    Paused --> Running : ResumeVM API call
    Running --> Shutting : ShutdownVM API call
    Shutting --> Shutdown : Graceful stop
    Shutting --> Failed : Shutdown timeout
    Running --> Rebooting : RebootVM API call
    Rebooting --> Running : Reboot complete
    Rebooting --> Failed : Reboot failure
    Created --> Deleting : DeleteVM API call
    Running --> Deleting : DeleteVM API call
    Paused --> Deleting : DeleteVM API call
    Shutdown --> Deleting : DeleteVM API call
    Deleting --> Deleted : Process terminated
    Failed --> Deleting : DeleteVM API call
    Deleted --> [*]
```

## Complete VM Creation Flow

```mermaid
flowchart TD
    Start([API: CreateVM]) --> GenID[Generate VM ID]
    GenID --> CheckProc{Process exists<br/>for VM?}
    
    CheckProc -->|No| CheckMax{Max processes<br/>reached?}
    CheckMax -->|Yes| Error1[Error: Too many processes]
    CheckMax -->|No| SpawnProc[Spawn Firecracker Process]
    
    SpawnProc --> CreateSocket[Create Unix Socket]
    CreateSocket --> WaitSocket[Wait for Socket Ready]
    WaitSocket --> SocketTimeout{Socket ready<br/>within 30s?}
    SocketTimeout -->|No| KillProc[Kill Process] --> Error2[Error: Startup timeout]
    SocketTimeout -->|Yes| ProcReady[Process Status: Ready]
    
    CheckProc -->|Yes| ProcReady
    ProcReady --> AssignVM[Assign VM to Process]
    AssignVM --> SetBusy[Process Status: Busy]
    SetBusy --> CreateClient[Create HTTP Client]
    CreateClient --> ConfigMachine[PUT /machine-config]
    
    ConfigMachine --> ConfigBoot[PUT /boot-source]
    ConfigBoot --> ConfigDrive[PUT /drives/rootfs]
    ConfigDrive --> ConfigMetrics[PUT /metrics + FIFO setup]
    ConfigMetrics --> StartFIFO[Start FIFO Reader Goroutine]
    StartFIFO --> RegisterVM[Register VM in Registry]
    RegisterVM --> Success([Return VM ID])
    
    Error1 --> End([End])
    Error2 --> End
    Success --> End
    
    style SpawnProc fill:#e1f5fe
    style ConfigMetrics fill:#f3e5f5
    style StartFIFO fill:#f3e5f5
    style Success fill:#e8f5e8
    style Error1 fill:#ffebee
    style Error2 fill:#ffebee
```

## VM Deletion Flow

```mermaid
flowchart TD
    Start([API: DeleteVM]) --> LookupVM{VM exists in<br/>registry?}
    LookupVM -->|No| Error1[Error: VM not found]
    LookupVM -->|Yes| GetClient[Get VM's Firecracker Client]
    
    GetClient --> CleanupFIFO[Close FIFO Channel]
    CleanupFIFO --> RemoveFIFO[Remove FIFO File]
    RemoveFIFO --> ReleaseProc[Release Process]
    
    ReleaseProc --> FindProc{Find process<br/>for VM?}
    FindProc -->|No| Error2[Error: Process not found]
    FindProc -->|Yes| CheckModel{Process reuse<br/>or 1:1 model?}
    
    CheckModel -->|Reuse| MarkReady[Mark Process Ready]
    CheckModel -->|1:1| TermProc[Terminate Process]
    
    TermProc --> SendSIGTERM[Send SIGTERM]
    SendSIGTERM --> WaitExit[Wait for Exit (5s)]
    WaitExit --> ExitTimeout{Process exited<br/>gracefully?}
    
    ExitTimeout -->|Yes| CleanupFiles[Remove Socket & PID Files]
    ExitTimeout -->|No| SendSIGKILL[Send SIGKILL]
    SendSIGKILL --> ForceWait[Wait for Force Exit]
    ForceWait --> CleanupFiles
    
    CleanupFiles --> RemoveFromRegistry[Remove from Process Registry]
    MarkReady --> UnregisterVM[Remove VM from Registry]
    RemoveFromRegistry --> UnregisterVM
    UnregisterVM --> Success([VM Deleted])
    
    Error1 --> End([End])
    Error2 --> End
    Success --> End
    
    style CleanupFIFO fill:#fff3e0
    style TermProc fill:#ffebee
    style SendSIGKILL fill:#ff5722,color:#fff
    style Success fill:#e8f5e8
    style Error1 fill:#ffebee
    style Error2 fill:#ffebee
```

## Process Monitoring Flow

```mermaid
flowchart TD
    Start([Process Created]) --> SpawnMonitor[Spawn Monitor Goroutine]
    SpawnMonitor --> WaitExit[Process.Wait()]
    WaitExit --> ProcessExit[Process Exited]
    ProcessExit --> LockRegistry[Lock Process Registry]
    
    LockRegistry --> StillExists{Process still in<br/>registry?}
    StillExists -->|No| UnlockExit[Unlock & Exit]
    StillExists -->|Yes| CheckError{Exit with<br/>error?}
    
    CheckError -->|Yes| LogError[Log Error + Exit Code]
    CheckError -->|No| LogExit[Log Unexpected Exit]
    
    LogError --> SetErrorStatus[Status: Error]
    LogExit --> SetStoppedStatus[Status: Stopped]
    
    SetErrorStatus --> CleanupSocket[Remove Socket File]
    SetStoppedStatus --> CleanupSocket
    CleanupSocket --> CleanupPID[Remove PID File]
    CleanupPID --> RemoveRegistry[Remove from Registry]
    RemoveRegistry --> UnlockRegistry[Unlock Registry]
    UnlockRegistry --> LogCleanup[Log Cleanup Complete]
    LogCleanup --> End([Monitor Exit])
    
    UnlockExit --> End
    
    style WaitExit fill:#e3f2fd
    style LogError fill:#ffebee
    style LogExit fill:#fff3e0
    style CleanupSocket fill:#f3e5f5
    style End fill:#e8f5e8
```

## FIFO Metrics Collection Flow

```mermaid
flowchart TD
    Start([VM Configuration]) --> CreateFIFO[Create Named Pipe<br/>syscall.Mkfifo()]
    CreateFIFO --> ConfigFC[Configure Firecracker<br/>PUT /metrics]
    ConfigFC --> InitChannel[Initialize Buffered Channel<br/>100 metrics capacity]
    InitChannel --> SpawnReader[Spawn FIFO Reader Goroutine]
    
    SpawnReader --> OpenFIFO[Open FIFO for Reading<br/>Blocks until FC connects]
    OpenFIFO --> FCConnected[Firecracker Connected]
    FCConnected --> JsonDecoder[Create JSON Decoder]
    
    JsonDecoder --> ReadLoop{Read JSON Line}
    ReadLoop --> DecodeMetrics[Decode Firecracker Metrics]
    DecodeMetrics --> ConvertMetrics[Convert to Standard Format]
    ConvertMetrics --> CacheMetrics[Cache Last Known Metrics]
    CacheMetrics --> SendChannel{Channel<br/>Available?}
    
    SendChannel -->|Yes| SendSuccess[Send to Channel]
    SendChannel -->|No| DropOldest[Drop Oldest Metric]
    DropOldest --> RetrySend[Retry Send]
    RetrySend --> SendSuccess
    SendSuccess --> ReadLoop
    
    ReadLoop -->|EOF| FCDisconnected[Firecracker Disconnected]
    ReadLoop -->|JSON Error| LogError[Log Parse Error]
    LogError --> ReadLoop
    
    FCDisconnected --> CloseChannel[Close Metrics Channel]
    CloseChannel --> RemoveFIFO[Remove FIFO File]
    RemoveFIFO --> CleanupCache[Clear Cached Metrics]
    CleanupCache --> End([FIFO Collection Stopped])
    
    style CreateFIFO fill:#e1f5fe
    style OpenFIFO fill:#fff3e0
    style SendChannel fill:#f3e5f5
    style DropOldest fill:#ffebee
    style End fill:#e8f5e8
```

## Context Management Flow

```mermaid
flowchart TD
    Start([API Request with Context]) --> ExtractTracing[Extract Tracing Info<br/>Span, Baggage, Tenant ID]
    ExtractTracing --> CreateProcessCtx[Create Process Context]
    
    CreateProcessCtx --> UseAppCtx[Base: Application Context<br/>Long-lived, survives requests]
    UseAppCtx --> CopySpan[Copy Trace Span<br/>For distributed tracing]
    CopySpan --> CopyBaggage[Copy Baggage<br/>tenant_id, user_id]
    CopyBaggage --> AddProcessMeta[Add Process Metadata<br/>process_id, component]
    
    AddProcessMeta --> LogTenant{Tenant context<br/>present?}
    LogTenant -->|Yes| LogAudit[Log Tenant Assignment<br/>For security audit]
    LogTenant -->|No| SpawnProcess[Spawn Firecracker Process]
    LogAudit --> SpawnProcess
    
    SpawnProcess --> ProcessLongevity[Process Runs Independent<br/>of Request Lifecycle]
    ProcessLongevity --> TraceCorrelation[Traces Correlated Across<br/>Request Boundaries]
    TraceCorrelation --> TenantIsolation[Tenant Context Preserved<br/>in Long-lived Process]
    TenantIsolation --> End([Context Bridge Complete])
    
    style ExtractTracing fill:#e3f2fd
    style UseAppCtx fill:#f3e5f5
    style CopySpan fill:#e8f5e8
    style CopyBaggage fill:#e8f5e8
    style LogAudit fill:#fff3e0
    style ProcessLongevity fill:#e1f5fe
    style End fill:#e8f5e8
```

## Error Recovery Patterns

### Process Failure Recovery

```mermaid
flowchart TD
    Start([Process Failure Detected]) --> IdentifyFailure{Failure Type?}
    
    IdentifyFailure -->|Crash| ProcessCrashed[Process Monitor Detects Exit]
    IdentifyFailure -->|Socket Error| SocketFailure[HTTP Client Error]
    IdentifyFailure -->|Resource Error| ResourceFailure[Creation Failed]
    
    ProcessCrashed --> LogCrash[Log Crash + Exit Code]
    SocketFailure --> LogSocket[Log Socket Error]
    ResourceFailure --> LogResource[Log Resource Error]
    
    LogCrash --> CleanupCrash[Cleanup: Socket, PID, Registry]
    LogSocket --> CleanupSocket[Cleanup: Client Connection]
    LogResource --> CleanupResource[Cleanup: Partial Process]
    
    CleanupCrash --> NotifyVM[Mark VM as Failed]
    CleanupSocket --> RetryConnection{Retry Connection?}
    CleanupResource --> ReturnError[Return Creation Error]
    
    RetryConnection -->|Yes| CreateNewClient[Create New Client]
    RetryConnection -->|No| NotifyVM
    CreateNewClient --> TestConnection[Test New Connection]
    TestConnection --> Success{Connection OK?}
    Success -->|Yes| RestoreService[Service Restored]
    Success -->|No| NotifyVM
    
    NotifyVM --> APIError[Propagate Error to API]
    ReturnError --> APIError
    RestoreService --> End([Recovery Complete])
    APIError --> End
    
    style ProcessCrashed fill:#ffebee
    style LogCrash fill:#ffebee
    style RestoreService fill:#e8f5e8
    style End fill:#e8f5e8
```

### Resource Cleanup Patterns

```mermaid
flowchart TD
    Start([Cleanup Trigger]) --> TriggerType{Trigger Type?}
    
    TriggerType -->|Normal Shutdown| GracefulCleanup[Graceful Cleanup]
    TriggerType -->|Process Crash| CrashCleanup[Crash Cleanup]
    TriggerType -->|Service Shutdown| ServiceCleanup[Service Shutdown]
    
    GracefulCleanup --> CloseVMConnections[Close VM API Connections]
    CrashCleanup --> DetectOrphans[Detect Orphaned Resources]
    ServiceCleanup --> StopAllProcesses[Stop All Processes]
    
    CloseVMConnections --> CloseFIFO[Close FIFO Channels]
    DetectOrphans --> CloseFIFO
    StopAllProcesses --> CloseFIFO
    
    CloseFIFO --> RemoveFIFOFiles[Remove FIFO Files]
    RemoveFIFOFiles --> CloseSocket[Remove Socket Files]
    CloseSocket --> RemovePIDFiles[Remove PID Files]
    RemovePIDFiles --> ClearRegistry[Clear Process Registry]
    ClearRegistry --> LogCompletion[Log Cleanup Complete]
    LogCompletion --> End([Cleanup Complete])
    
    style CrashCleanup fill:#ffebee
    style StopAllProcesses fill:#fff3e0
    style End fill:#e8f5e8
```

This comprehensive flow documentation covers all major operational patterns in the Firecracker process manager, from normal operations to failure recovery scenarios.