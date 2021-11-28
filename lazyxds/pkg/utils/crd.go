// Copyright Aeraki Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"bytes"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
)

// BuildPatchStruct creates types.Struct from istio EnvoyFilter slice string
func BuildPatchStruct(config string) (*types.Struct, error) {
	m := jsonpb.Unmarshaler{}

	out := &types.Struct{}
	err := m.Unmarshal(bytes.NewReader([]byte(config)), out)
	if err != nil {
		return nil, err
	}

	return out, nil
}
