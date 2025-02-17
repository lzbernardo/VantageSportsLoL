// Code generated by protoc-gen-go.
// source: google.golang.org/genproto/googleapis/iam/v1/policy.proto
// DO NOT EDIT!

package google_iam_v1

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// Defines an Identity and Access Management (IAM) policy. It is used to
// specify access control policies for Cloud Platform resources.
//
//
// A `Policy` consists of a list of `bindings`. A `Binding` binds a list of
// `members` to a `role`, where the members can be user accounts, Google groups,
// Google domains, and service accounts. A `role` is a named list of permissions
// defined by IAM.
//
// **Example**
//
//     {
//       "bindings": [
//         {
//           "role": "roles/owner",
//           "members": [
//             "user:mike@example.com",
//             "group:admins@example.com",
//             "domain:google.com",
//             "serviceAccount:my-other-app@appspot.gserviceaccount.com",
//           ]
//         },
//         {
//           "role": "roles/viewer",
//           "members": ["user:sean@example.com"]
//         }
//       ]
//     }
//
// For a description of IAM and its features, see the
// [IAM developer's guide](https://cloud.google.com/iam).
type Policy struct {
	// Version of the `Policy`. The default version is 0.
	Version int32 `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	// Associates a list of `members` to a `role`.
	// Multiple `bindings` must not be specified for the same `role`.
	// `bindings` with no members will result in an error.
	Bindings []*Binding `protobuf:"bytes,4,rep,name=bindings" json:"bindings,omitempty"`
	// `etag` is used for optimistic concurrency control as a way to help
	// prevent simultaneous updates of a policy from overwriting each other.
	// It is strongly suggested that systems make use of the `etag` in the
	// read-modify-write cycle to perform policy updates in order to avoid race
	// conditions: An `etag` is returned in the response to `getIamPolicy`, and
	// systems are expected to put that etag in the request to `setIamPolicy` to
	// ensure that their change will be applied to the same version of the policy.
	//
	// If no `etag` is provided in the call to `setIamPolicy`, then the existing
	// policy is overwritten blindly.
	Etag []byte `protobuf:"bytes,3,opt,name=etag,proto3" json:"etag,omitempty"`
}

func (m *Policy) Reset()                    { *m = Policy{} }
func (m *Policy) String() string            { return proto.CompactTextString(m) }
func (*Policy) ProtoMessage()               {}
func (*Policy) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{0} }

func (m *Policy) GetBindings() []*Binding {
	if m != nil {
		return m.Bindings
	}
	return nil
}

// Associates `members` with a `role`.
type Binding struct {
	// Role that is assigned to `members`.
	// For example, `roles/viewer`, `roles/editor`, or `roles/owner`.
	// Required
	Role string `protobuf:"bytes,1,opt,name=role" json:"role,omitempty"`
	// Specifies the identities requesting access for a Cloud Platform resource.
	// `members` can have the following values:
	//
	// * `allUsers`: A special identifier that represents anyone who is
	//    on the internet; with or without a Google account.
	//
	// * `allAuthenticatedUsers`: A special identifier that represents anyone
	//    who is authenticated with a Google account or a service account.
	//
	// * `user:{emailid}`: An email address that represents a specific Google
	//    account. For example, `alice@gmail.com` or `joe@example.com`.
	//
	// * `serviceAccount:{emailid}`: An email address that represents a service
	//    account. For example, `my-other-app@appspot.gserviceaccount.com`.
	//
	// * `group:{emailid}`: An email address that represents a Google group.
	//    For example, `admins@example.com`.
	//
	// * `domain:{domain}`: A Google Apps domain name that represents all the
	//    users of that domain. For example, `google.com` or `example.com`.
	//
	//
	Members []string `protobuf:"bytes,2,rep,name=members" json:"members,omitempty"`
}

func (m *Binding) Reset()                    { *m = Binding{} }
func (m *Binding) String() string            { return proto.CompactTextString(m) }
func (*Binding) ProtoMessage()               {}
func (*Binding) Descriptor() ([]byte, []int) { return fileDescriptor1, []int{1} }

func init() {
	proto.RegisterType((*Policy)(nil), "google.iam.v1.Policy")
	proto.RegisterType((*Binding)(nil), "google.iam.v1.Binding")
}

func init() {
	proto.RegisterFile("google.golang.org/genproto/googleapis/iam/v1/policy.proto", fileDescriptor1)
}

var fileDescriptor1 = []byte{
	// 216 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x54, 0x8f, 0xc1, 0x6a, 0x03, 0x21,
	0x18, 0x84, 0xb1, 0x9b, 0x26, 0x8d, 0x69, 0x0f, 0xf5, 0x50, 0x3c, 0x4a, 0xa0, 0xe0, 0x49, 0x49,
	0x7a, 0x28, 0xbd, 0xee, 0x13, 0x2c, 0xbe, 0x81, 0x9b, 0xca, 0x8f, 0x45, 0xfd, 0x17, 0x0d, 0x0b,
	0x7d, 0xf3, 0x1e, 0xcb, 0xba, 0xd9, 0xc0, 0xde, 0xfe, 0xe1, 0x1b, 0x9d, 0x19, 0xfa, 0x05, 0x88,
	0x10, 0x9c, 0x02, 0x0c, 0x36, 0x81, 0xc2, 0x0c, 0x1a, 0x5c, 0x1a, 0x32, 0x5e, 0x51, 0xcf, 0xc8,
	0x0e, 0xbe, 0x68, 0x6f, 0xa3, 0x1e, 0x4f, 0x7a, 0xc0, 0xe0, 0x2f, 0xbf, 0xaa, 0x62, 0xf6, 0x72,
	0x7b, 0xea, 0x6d, 0x54, 0xe3, 0xe9, 0xf8, 0x43, 0xb7, 0x5d, 0xc5, 0x8c, 0xd3, 0xdd, 0xe8, 0x72,
	0xf1, 0x98, 0x38, 0x11, 0x44, 0x3e, 0x9a, 0x45, 0xb2, 0x33, 0x7d, 0xea, 0x7d, 0xfa, 0xf6, 0x09,
	0x0a, 0xdf, 0x88, 0x46, 0x1e, 0xce, 0x6f, 0x6a, 0xf5, 0x8b, 0x6a, 0x67, 0x6c, 0xee, 0x3e, 0xc6,
	0xe8, 0xc6, 0x5d, 0x2d, 0xf0, 0x46, 0x10, 0xf9, 0x6c, 0xea, 0x7d, 0xfc, 0xa4, 0xbb, 0x9b, 0x71,
	0xc2, 0x19, 0x83, 0xab, 0x49, 0x7b, 0x53, 0xef, 0xa9, 0x40, 0x74, 0xb1, 0x77, 0xb9, 0xf0, 0x07,
	0xd1, 0xc8, 0xbd, 0x59, 0x64, 0xfb, 0x4e, 0x5f, 0x2f, 0x18, 0xd7, 0x99, 0xed, 0x61, 0xee, 0xdd,
	0x4d, 0xab, 0x3a, 0xf2, 0x47, 0x48, 0xbf, 0xad, 0x0b, 0x3f, 0xfe, 0x03, 0x00, 0x00, 0xff, 0xff,
	0xcc, 0x5d, 0xa8, 0xf7, 0x1e, 0x01, 0x00, 0x00,
}
