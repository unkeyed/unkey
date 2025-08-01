// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: proto/ctrl/v1/openapi.proto

package ctrlv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GetOpenApiDiffRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	OldVersionId  string                 `protobuf:"bytes,1,opt,name=old_version_id,json=oldVersionId,proto3" json:"old_version_id,omitempty"`
	NewVersionId  string                 `protobuf:"bytes,2,opt,name=new_version_id,json=newVersionId,proto3" json:"new_version_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetOpenApiDiffRequest) Reset() {
	*x = GetOpenApiDiffRequest{}
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetOpenApiDiffRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetOpenApiDiffRequest) ProtoMessage() {}

func (x *GetOpenApiDiffRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetOpenApiDiffRequest.ProtoReflect.Descriptor instead.
func (*GetOpenApiDiffRequest) Descriptor() ([]byte, []int) {
	return file_proto_ctrl_v1_openapi_proto_rawDescGZIP(), []int{0}
}

func (x *GetOpenApiDiffRequest) GetOldVersionId() string {
	if x != nil {
		return x.OldVersionId
	}
	return ""
}

func (x *GetOpenApiDiffRequest) GetNewVersionId() string {
	if x != nil {
		return x.NewVersionId
	}
	return ""
}

type ChangelogEntry struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Text          string                 `protobuf:"bytes,2,opt,name=text,proto3" json:"text,omitempty"`
	Level         int32                  `protobuf:"varint,3,opt,name=level,proto3" json:"level,omitempty"`
	Operation     string                 `protobuf:"bytes,4,opt,name=operation,proto3" json:"operation,omitempty"`
	OperationId   *string                `protobuf:"bytes,5,opt,name=operation_id,json=operationId,proto3,oneof" json:"operation_id,omitempty"`
	Path          string                 `protobuf:"bytes,6,opt,name=path,proto3" json:"path,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ChangelogEntry) Reset() {
	*x = ChangelogEntry{}
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ChangelogEntry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ChangelogEntry) ProtoMessage() {}

func (x *ChangelogEntry) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ChangelogEntry.ProtoReflect.Descriptor instead.
func (*ChangelogEntry) Descriptor() ([]byte, []int) {
	return file_proto_ctrl_v1_openapi_proto_rawDescGZIP(), []int{1}
}

func (x *ChangelogEntry) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ChangelogEntry) GetText() string {
	if x != nil {
		return x.Text
	}
	return ""
}

func (x *ChangelogEntry) GetLevel() int32 {
	if x != nil {
		return x.Level
	}
	return 0
}

func (x *ChangelogEntry) GetOperation() string {
	if x != nil {
		return x.Operation
	}
	return ""
}

func (x *ChangelogEntry) GetOperationId() string {
	if x != nil && x.OperationId != nil {
		return *x.OperationId
	}
	return ""
}

func (x *ChangelogEntry) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

type DiffSummary struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Diff          bool                   `protobuf:"varint,1,opt,name=diff,proto3" json:"diff,omitempty"`
	Details       *DiffDetails           `protobuf:"bytes,2,opt,name=details,proto3" json:"details,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DiffSummary) Reset() {
	*x = DiffSummary{}
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DiffSummary) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DiffSummary) ProtoMessage() {}

func (x *DiffSummary) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DiffSummary.ProtoReflect.Descriptor instead.
func (*DiffSummary) Descriptor() ([]byte, []int) {
	return file_proto_ctrl_v1_openapi_proto_rawDescGZIP(), []int{2}
}

func (x *DiffSummary) GetDiff() bool {
	if x != nil {
		return x.Diff
	}
	return false
}

func (x *DiffSummary) GetDetails() *DiffDetails {
	if x != nil {
		return x.Details
	}
	return nil
}

type DiffDetails struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Endpoints     *DiffCounts            `protobuf:"bytes,1,opt,name=endpoints,proto3" json:"endpoints,omitempty"`
	Paths         *DiffCounts            `protobuf:"bytes,2,opt,name=paths,proto3" json:"paths,omitempty"`
	Schemas       *DiffCounts            `protobuf:"bytes,3,opt,name=schemas,proto3" json:"schemas,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DiffDetails) Reset() {
	*x = DiffDetails{}
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DiffDetails) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DiffDetails) ProtoMessage() {}

func (x *DiffDetails) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DiffDetails.ProtoReflect.Descriptor instead.
func (*DiffDetails) Descriptor() ([]byte, []int) {
	return file_proto_ctrl_v1_openapi_proto_rawDescGZIP(), []int{3}
}

func (x *DiffDetails) GetEndpoints() *DiffCounts {
	if x != nil {
		return x.Endpoints
	}
	return nil
}

func (x *DiffDetails) GetPaths() *DiffCounts {
	if x != nil {
		return x.Paths
	}
	return nil
}

func (x *DiffDetails) GetSchemas() *DiffCounts {
	if x != nil {
		return x.Schemas
	}
	return nil
}

type DiffCounts struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Added         int32                  `protobuf:"varint,1,opt,name=added,proto3" json:"added,omitempty"`
	Deleted       int32                  `protobuf:"varint,2,opt,name=deleted,proto3" json:"deleted,omitempty"`
	Modified      int32                  `protobuf:"varint,3,opt,name=modified,proto3" json:"modified,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DiffCounts) Reset() {
	*x = DiffCounts{}
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DiffCounts) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DiffCounts) ProtoMessage() {}

func (x *DiffCounts) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DiffCounts.ProtoReflect.Descriptor instead.
func (*DiffCounts) Descriptor() ([]byte, []int) {
	return file_proto_ctrl_v1_openapi_proto_rawDescGZIP(), []int{4}
}

func (x *DiffCounts) GetAdded() int32 {
	if x != nil {
		return x.Added
	}
	return 0
}

func (x *DiffCounts) GetDeleted() int32 {
	if x != nil {
		return x.Deleted
	}
	return 0
}

func (x *DiffCounts) GetModified() int32 {
	if x != nil {
		return x.Modified
	}
	return 0
}

type GetOpenApiDiffResponse struct {
	state              protoimpl.MessageState `protogen:"open.v1"`
	Summary            *DiffSummary           `protobuf:"bytes,1,opt,name=summary,proto3" json:"summary,omitempty"`
	HasBreakingChanges bool                   `protobuf:"varint,2,opt,name=has_breaking_changes,json=hasBreakingChanges,proto3" json:"has_breaking_changes,omitempty"`
	Changes            []*ChangelogEntry      `protobuf:"bytes,3,rep,name=changes,proto3" json:"changes,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *GetOpenApiDiffResponse) Reset() {
	*x = GetOpenApiDiffResponse{}
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetOpenApiDiffResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetOpenApiDiffResponse) ProtoMessage() {}

func (x *GetOpenApiDiffResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_ctrl_v1_openapi_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetOpenApiDiffResponse.ProtoReflect.Descriptor instead.
func (*GetOpenApiDiffResponse) Descriptor() ([]byte, []int) {
	return file_proto_ctrl_v1_openapi_proto_rawDescGZIP(), []int{5}
}

func (x *GetOpenApiDiffResponse) GetSummary() *DiffSummary {
	if x != nil {
		return x.Summary
	}
	return nil
}

func (x *GetOpenApiDiffResponse) GetHasBreakingChanges() bool {
	if x != nil {
		return x.HasBreakingChanges
	}
	return false
}

func (x *GetOpenApiDiffResponse) GetChanges() []*ChangelogEntry {
	if x != nil {
		return x.Changes
	}
	return nil
}

var File_proto_ctrl_v1_openapi_proto protoreflect.FileDescriptor

const file_proto_ctrl_v1_openapi_proto_rawDesc = "" +
	"\n" +
	"\x1bproto/ctrl/v1/openapi.proto\x12\actrl.v1\"c\n" +
	"\x15GetOpenApiDiffRequest\x12$\n" +
	"\x0eold_version_id\x18\x01 \x01(\tR\foldVersionId\x12$\n" +
	"\x0enew_version_id\x18\x02 \x01(\tR\fnewVersionId\"\xb5\x01\n" +
	"\x0eChangelogEntry\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12\x12\n" +
	"\x04text\x18\x02 \x01(\tR\x04text\x12\x14\n" +
	"\x05level\x18\x03 \x01(\x05R\x05level\x12\x1c\n" +
	"\toperation\x18\x04 \x01(\tR\toperation\x12&\n" +
	"\foperation_id\x18\x05 \x01(\tH\x00R\voperationId\x88\x01\x01\x12\x12\n" +
	"\x04path\x18\x06 \x01(\tR\x04pathB\x0f\n" +
	"\r_operation_id\"Q\n" +
	"\vDiffSummary\x12\x12\n" +
	"\x04diff\x18\x01 \x01(\bR\x04diff\x12.\n" +
	"\adetails\x18\x02 \x01(\v2\x14.ctrl.v1.DiffDetailsR\adetails\"\x9a\x01\n" +
	"\vDiffDetails\x121\n" +
	"\tendpoints\x18\x01 \x01(\v2\x13.ctrl.v1.DiffCountsR\tendpoints\x12)\n" +
	"\x05paths\x18\x02 \x01(\v2\x13.ctrl.v1.DiffCountsR\x05paths\x12-\n" +
	"\aschemas\x18\x03 \x01(\v2\x13.ctrl.v1.DiffCountsR\aschemas\"X\n" +
	"\n" +
	"DiffCounts\x12\x14\n" +
	"\x05added\x18\x01 \x01(\x05R\x05added\x12\x18\n" +
	"\adeleted\x18\x02 \x01(\x05R\adeleted\x12\x1a\n" +
	"\bmodified\x18\x03 \x01(\x05R\bmodified\"\xad\x01\n" +
	"\x16GetOpenApiDiffResponse\x12.\n" +
	"\asummary\x18\x01 \x01(\v2\x14.ctrl.v1.DiffSummaryR\asummary\x120\n" +
	"\x14has_breaking_changes\x18\x02 \x01(\bR\x12hasBreakingChanges\x121\n" +
	"\achanges\x18\x03 \x03(\v2\x17.ctrl.v1.ChangelogEntryR\achanges2e\n" +
	"\x0eOpenApiService\x12S\n" +
	"\x0eGetOpenApiDiff\x12\x1e.ctrl.v1.GetOpenApiDiffRequest\x1a\x1f.ctrl.v1.GetOpenApiDiffResponse\"\x00B6Z4github.com/unkeyed/unkey/go/gen/proto/ctrl/v1;ctrlv1b\x06proto3"

var (
	file_proto_ctrl_v1_openapi_proto_rawDescOnce sync.Once
	file_proto_ctrl_v1_openapi_proto_rawDescData []byte
)

func file_proto_ctrl_v1_openapi_proto_rawDescGZIP() []byte {
	file_proto_ctrl_v1_openapi_proto_rawDescOnce.Do(func() {
		file_proto_ctrl_v1_openapi_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_proto_ctrl_v1_openapi_proto_rawDesc), len(file_proto_ctrl_v1_openapi_proto_rawDesc)))
	})
	return file_proto_ctrl_v1_openapi_proto_rawDescData
}

var file_proto_ctrl_v1_openapi_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_proto_ctrl_v1_openapi_proto_goTypes = []any{
	(*GetOpenApiDiffRequest)(nil),  // 0: ctrl.v1.GetOpenApiDiffRequest
	(*ChangelogEntry)(nil),         // 1: ctrl.v1.ChangelogEntry
	(*DiffSummary)(nil),            // 2: ctrl.v1.DiffSummary
	(*DiffDetails)(nil),            // 3: ctrl.v1.DiffDetails
	(*DiffCounts)(nil),             // 4: ctrl.v1.DiffCounts
	(*GetOpenApiDiffResponse)(nil), // 5: ctrl.v1.GetOpenApiDiffResponse
}
var file_proto_ctrl_v1_openapi_proto_depIdxs = []int32{
	3, // 0: ctrl.v1.DiffSummary.details:type_name -> ctrl.v1.DiffDetails
	4, // 1: ctrl.v1.DiffDetails.endpoints:type_name -> ctrl.v1.DiffCounts
	4, // 2: ctrl.v1.DiffDetails.paths:type_name -> ctrl.v1.DiffCounts
	4, // 3: ctrl.v1.DiffDetails.schemas:type_name -> ctrl.v1.DiffCounts
	2, // 4: ctrl.v1.GetOpenApiDiffResponse.summary:type_name -> ctrl.v1.DiffSummary
	1, // 5: ctrl.v1.GetOpenApiDiffResponse.changes:type_name -> ctrl.v1.ChangelogEntry
	0, // 6: ctrl.v1.OpenApiService.GetOpenApiDiff:input_type -> ctrl.v1.GetOpenApiDiffRequest
	5, // 7: ctrl.v1.OpenApiService.GetOpenApiDiff:output_type -> ctrl.v1.GetOpenApiDiffResponse
	7, // [7:8] is the sub-list for method output_type
	6, // [6:7] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_proto_ctrl_v1_openapi_proto_init() }
func file_proto_ctrl_v1_openapi_proto_init() {
	if File_proto_ctrl_v1_openapi_proto != nil {
		return
	}
	file_proto_ctrl_v1_openapi_proto_msgTypes[1].OneofWrappers = []any{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_ctrl_v1_openapi_proto_rawDesc), len(file_proto_ctrl_v1_openapi_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_ctrl_v1_openapi_proto_goTypes,
		DependencyIndexes: file_proto_ctrl_v1_openapi_proto_depIdxs,
		MessageInfos:      file_proto_ctrl_v1_openapi_proto_msgTypes,
	}.Build()
	File_proto_ctrl_v1_openapi_proto = out.File
	file_proto_ctrl_v1_openapi_proto_goTypes = nil
	file_proto_ctrl_v1_openapi_proto_depIdxs = nil
}
