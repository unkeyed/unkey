# Railway-Style Pricing Model: AWS Metal + Firecracker

## Core Strategy
- **Hardware**: AWS R6I Metal - 128 vCPUs, 1024 GB RAM
- **Platform Overhead**: 10% reserved for management/monitoring
- **CPU Oversubscription**: 8x (922 sellable vCPUs after overhead)  
- **Memory Oversubscription**: None (922 GB sellable after overhead)
- **Virtualization**: Firecracker microVMs

## Pricing Model (Railway-Style)

**Rate Structure:**
- **Provisioned Memory**: $10/GB/month (flat rate)
- **Active CPU**: $20/vCPU/month (based on average usage)

**Formula**: `(provisioned_gb × $10) + (average_active_cpus × $20)`

## Customer Configuration Analysis

**Pricing for different customer types:**

| vCPUs | Memory (GB) | Avg CPU Usage | Active vCPUs | Memory Cost | CPU Cost | Monthly Cost | Use Case |
|-------|-------------|---------------|--------------|-------------|----------|--------------|----------|
| 1 | 1 | 20% | 0.2 | $10 | $4 | $14 | Small API, dev environment |
| 2 | 2 | 30% | 0.6 | $20 | $12 | $32 | Medium API, staging |
| 4 | 4 | 40% | 1.6 | $40 | $32 | $72 | Production API, web app |
| 8 | 8 | 50% | 4.0 | $80 | $80 | $160 | High-traffic API, background jobs |
| 16 | 16 | 60% | 9.6 | $160 | $192 | $352 | Enterprise API, data processing |
| 2 | 8 | 15% | 0.3 | $80 | $6 | $86 | Memory-heavy cache, database |
| 8 | 4 | 70% | 5.6 | $40 | $112 | $152 | CPU-intensive processing |

## Revenue Analysis per R6I Machine

**Maximum density scenarios:**

### Scenario A: All Small Customers (1 vCPU + 1 GB)
- **Max customers**: 922 (limited by memory)
- **Monthly revenue**: 922 × $14 = $12,908
- **Hardware cost**: $3,894 (reserved)
- **Profit**: $9,014 (70% margin)
- **CPU utilization**: 184 active vCPUs (20% of total capacity)

### Scenario B: Mixed Customer Base
- **200 small (1v+1g)**: $2,800 revenue, 200 GB memory
- **150 medium (2v+2g)**: $4,800 revenue, 300 GB memory  
- **100 production (4v+4g)**: $7,200 revenue, 400 GB memory
- **Total**: 450 customers, 900 GB memory, $14,800 revenue
- **Profit**: $10,906 (74% margin)

### Scenario C: Enterprise Focus (Higher-spec customers)
- **58 enterprise (16v+16g)**: 928 GB memory, $20,416 revenue
- **Profit**: $16,522 (81% margin)
- **CPU utilization**: 557 active vCPUs (60% of total capacity)

## Scale to $1M ARR

**Target**: $83,333 monthly revenue

| Customer Mix | Machines Needed | Infrastructure Cost | Monthly Profit | Margin |
|--------------|----------------|-------------------|----------------|---------|
| **Small focus** | 7 machines | $27,258 | $56,075 | 67% |
| **Mixed base** | 6 machines | $23,364 | $59,969 | 72% |
| **Enterprise focus** | 5 machines | $19,470 | $63,863 | 77% |

## Competitive Advantage

**Railway pricing gives you massive margins:**
- **70-81% profit margins** across all scenarios
- **Break-even at just 28% capacity** (1 customer per 3.5 GB memory)
- **Much higher revenue per customer** than cost-recovery model

## Key Insights

- **Memory pricing drives profitability** - $10/GB vs $4.22 hardware cost
- **CPU becomes bonus revenue** - any usage above break-even is profit
- **Enterprise customers most profitable** - higher CPU usage, premium pricing
- **Can reach $1M ARR with just 5-7 machines** depending on customer mix
- **Railway's pricing model is extremely profitable** for infrastructure providers

## Risk Analysis

**Advantages:**
- **Massive profit margins** on every customer
- **Simple, predictable pricing** 
- **Higher revenue per machine** than cost-recovery model

**Risks:**
- **Price sensitivity** - customers may shop around
- **Lower customer density** - fewer total customers per machine
- **Competitive pressure** - other providers may undercut pricing

**Recommendation**: Railway pricing offers much higher margins but may limit customer acquisition due to premium pricing.