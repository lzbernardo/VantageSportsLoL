package users

import proto "github.com/golang/protobuf/proto"

// We write our own User and LoginRequest String() methods here to ensure that
// the password fields don't accidentally get logged in the RPC trace logs.
//
// When the protocol buffer generated code is re-generated, you'll need to
// manually delete these two methods from the generated code in users.pb.go.

func (m *User) String() string {
	if m == nil {
		return "<nil>"
	}
	orig := m.Password
	m.Password = "<not shown>"

	str := proto.CompactTextString(m)

	m.Password = orig
	return str
}

// String does not log the password (in traces, which would be bad)
func (m *LoginRequest) String() string {
	orig := m.Password
	m.Password = "<not shown>"

	str := proto.CompactTextString(m)

	m.Password = orig
	return str
}
