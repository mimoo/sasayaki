// Code generated by protoc-gen-go. DO NOT EDIT.
// source: messages.proto

/*
Package serialization is a generated protocol buffer package.

It is generated from these files:
	messages.proto

It has these top-level messages:
	GetMessage
*/
package serialization

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type GetMessage_RequestType int32

const (
	GetMessage_GetMessages GetMessage_RequestType = 0
	GetMessage_GetContacts GetMessage_RequestType = 1
	GetMessage_GetProofs   GetMessage_RequestType = 2
)

var GetMessage_RequestType_name = map[int32]string{
	0: "GetMessages",
	1: "GetContacts",
	2: "GetProofs",
}
var GetMessage_RequestType_value = map[string]int32{
	"GetMessages": 0,
	"GetContacts": 1,
	"GetProofs":   2,
}

func (x GetMessage_RequestType) Enum() *GetMessage_RequestType {
	p := new(GetMessage_RequestType)
	*p = x
	return p
}
func (x GetMessage_RequestType) String() string {
	return proto.EnumName(GetMessage_RequestType_name, int32(x))
}
func (x *GetMessage_RequestType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(GetMessage_RequestType_value, data, "GetMessage_RequestType")
	if err != nil {
		return err
	}
	*x = GetMessage_RequestType(value)
	return nil
}
func (GetMessage_RequestType) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

type GetMessage struct {
	ClientName       *string               `protobuf:"bytes,1,req,name=clientName" json:"clientName,omitempty"`
	ClientId         *int32                `protobuf:"varint,2,req,name=clientId" json:"clientId,omitempty"`
	Description      *string               `protobuf:"bytes,3,opt,name=description,def=NONE" json:"description,omitempty"`
	Messageitems     []*GetMessage_MsgItem `protobuf:"bytes,4,rep,name=messageitems" json:"messageitems,omitempty"`
	XXX_unrecognized []byte                `json:"-"`
}

func (m *GetMessage) Reset()                    { *m = GetMessage{} }
func (m *GetMessage) String() string            { return proto.CompactTextString(m) }
func (*GetMessage) ProtoMessage()               {}
func (*GetMessage) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

const Default_GetMessage_Description string = "NONE"

func (m *GetMessage) GetClientName() string {
	if m != nil && m.ClientName != nil {
		return *m.ClientName
	}
	return ""
}

func (m *GetMessage) GetClientId() int32 {
	if m != nil && m.ClientId != nil {
		return *m.ClientId
	}
	return 0
}

func (m *GetMessage) GetDescription() string {
	if m != nil && m.Description != nil {
		return *m.Description
	}
	return Default_GetMessage_Description
}

func (m *GetMessage) GetMessageitems() []*GetMessage_MsgItem {
	if m != nil {
		return m.Messageitems
	}
	return nil
}

type GetMessage_MsgItem struct {
	Id               *int32                  `protobuf:"varint,1,req,name=id" json:"id,omitempty"`
	ItemName         *string                 `protobuf:"bytes,2,opt,name=itemName" json:"itemName,omitempty"`
	ItemValue        *int32                  `protobuf:"varint,3,opt,name=itemValue" json:"itemValue,omitempty"`
	RequestType      *GetMessage_RequestType `protobuf:"varint,4,opt,name=requestType,enum=serialization.GetMessage_RequestType" json:"requestType,omitempty"`
	XXX_unrecognized []byte                  `json:"-"`
}

func (m *GetMessage_MsgItem) Reset()                    { *m = GetMessage_MsgItem{} }
func (m *GetMessage_MsgItem) String() string            { return proto.CompactTextString(m) }
func (*GetMessage_MsgItem) ProtoMessage()               {}
func (*GetMessage_MsgItem) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

func (m *GetMessage_MsgItem) GetId() int32 {
	if m != nil && m.Id != nil {
		return *m.Id
	}
	return 0
}

func (m *GetMessage_MsgItem) GetItemName() string {
	if m != nil && m.ItemName != nil {
		return *m.ItemName
	}
	return ""
}

func (m *GetMessage_MsgItem) GetItemValue() int32 {
	if m != nil && m.ItemValue != nil {
		return *m.ItemValue
	}
	return 0
}

func (m *GetMessage_MsgItem) GetRequestType() GetMessage_RequestType {
	if m != nil && m.RequestType != nil {
		return *m.RequestType
	}
	return GetMessage_GetMessages
}

func init() {
	proto.RegisterType((*GetMessage)(nil), "serialization.GetMessage")
	proto.RegisterType((*GetMessage_MsgItem)(nil), "serialization.GetMessage.MsgItem")
	proto.RegisterEnum("serialization.GetMessage_RequestType", GetMessage_RequestType_name, GetMessage_RequestType_value)
}

func init() { proto.RegisterFile("messages.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 279 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x8f, 0x3f, 0x4f, 0xbb, 0x40,
	0x18, 0xc7, 0x7f, 0x1c, 0x90, 0x9f, 0x3c, 0x58, 0x24, 0x37, 0x91, 0xc6, 0x18, 0x6c, 0xa2, 0x61,
	0x62, 0xe8, 0xe8, 0xe0, 0x62, 0x1a, 0xd2, 0xa1, 0x68, 0x2e, 0xc6, 0xfd, 0x02, 0x8f, 0xcd, 0x25,
	0xc0, 0x21, 0xcf, 0x75, 0xd0, 0x17, 0xe2, 0xe4, 0x8b, 0x35, 0x50, 0x52, 0xe8, 0xe0, 0x76, 0xdf,
	0x7f, 0x97, 0xcf, 0x03, 0x41, 0x8d, 0x44, 0x72, 0x8f, 0x94, 0xb6, 0x9d, 0x36, 0x9a, 0x2f, 0x08,
	0x3b, 0x25, 0x2b, 0xf5, 0x25, 0x8d, 0xd2, 0xcd, 0xea, 0xdb, 0x06, 0xc8, 0xd0, 0xec, 0x8e, 0x25,
	0x7e, 0x03, 0x50, 0x54, 0x0a, 0x1b, 0x93, 0xcb, 0x1a, 0x23, 0x2b, 0x66, 0x89, 0x27, 0x66, 0x0e,
	0x5f, 0xc2, 0xc5, 0x51, 0x6d, 0xcb, 0x88, 0xc5, 0x2c, 0x71, 0xc5, 0x49, 0xf3, 0x7b, 0xf0, 0x4b,
	0xa4, 0xa2, 0x53, 0x6d, 0xff, 0x73, 0x64, 0xc7, 0x56, 0xe2, 0x3d, 0x38, 0xf9, 0x73, 0xbe, 0x11,
	0xf3, 0x80, 0x6f, 0xe0, 0x72, 0x64, 0x52, 0x06, 0x6b, 0x8a, 0x9c, 0xd8, 0x4e, 0xfc, 0xf5, 0x6d,
	0x7a, 0x06, 0x96, 0x4e, 0x50, 0xe9, 0x8e, 0xf6, 0x5b, 0x83, 0xb5, 0x38, 0x9b, 0x2d, 0x7f, 0x2c,
	0xf8, 0x3f, 0x26, 0x3c, 0x00, 0xa6, 0xca, 0x01, 0xd7, 0x15, 0x4c, 0x95, 0x3d, 0x66, 0x5f, 0x1a,
	0x8e, 0x60, 0x3d, 0x87, 0x38, 0x69, 0x7e, 0x0d, 0x5e, 0xff, 0x7e, 0x93, 0xd5, 0x01, 0x07, 0x48,
	0x57, 0x4c, 0x06, 0xcf, 0xc0, 0xef, 0xf0, 0xe3, 0x80, 0x64, 0x5e, 0x3f, 0x5b, 0x8c, 0x9c, 0xd8,
	0x4a, 0x82, 0xf5, 0xdd, 0xdf, 0x6c, 0x62, 0x2a, 0x8b, 0xf9, 0x72, 0xf5, 0x08, 0xfe, 0x2c, 0xe3,
	0x57, 0xe0, 0x4f, 0x2b, 0x0a, 0xff, 0x8d, 0xc6, 0x93, 0x6e, 0x8c, 0x2c, 0x0c, 0x85, 0x16, 0x5f,
	0x80, 0x97, 0xa1, 0x79, 0xe9, 0xb4, 0x7e, 0xa7, 0x90, 0xfd, 0x06, 0x00, 0x00, 0xff, 0xff, 0x74,
	0x29, 0x80, 0xcd, 0xb8, 0x01, 0x00, 0x00,
}