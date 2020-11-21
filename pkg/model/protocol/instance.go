package protocol

import "strings"

var cache map[string]Instance = make(map[string]Instance)

type Instance string

// Unsupported - value to signify that the protocol is unsupported.
const Unsupported Instance = "UnsupportedProtocol"

func Parse(s string) Instance {
	name := strings.ToLower(s)
	if protocol, ok := cache[name]; ok {
		return protocol
	}

	var protocol Instance = Instance(name)
	cache[name] = protocol
	return protocol
}

func GetLayer7ProtocolFromPortName(name string) Instance {
	s := strings.Split(name, "-")
	if len(s) > 1 {
		return Parse(s[1])
	}
	return Unsupported
}
