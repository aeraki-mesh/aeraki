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

package kube

import (
	"bytes"
	"os"
	"text/template"

	"istio.io/pkg/log"
)

var bootstrapConfig = `
    {
       "static_resources":{
          "clusters":[
             {
                "name":"aeraki-xds",
                "type":"STRICT_DNS",
                "connect_timeout":"1s",
                "max_requests_per_connection":1,
                "circuit_breakers":{
                   "thresholds":[
                      {
                         "max_connections":100000,
                         "max_pending_requests":100000,
                         "max_requests":100000
                      },
                      {
                         "priority":"HIGH",
                         "max_connections":100000,
                         "max_pending_requests":100000,
                         "max_requests":100000
                      }
                   ]
                },
                "http2_protocol_options":{

                },
                "upstream_connection_options":{
                   "tcp_keepalive":{
                      "keepalive_time":300
                   }
                },
                "load_assignment":{
                   "cluster_name":"aeraki-xds",
                   "endpoints":[
                      {
                         "lb_endpoints":[
                            {
                               "endpoint":{
                                  "address":{
                                     "socket_address":{
                                        "address":"{{.Address}}",
                                        "port_value"{{.PortValue}}
                                     }
                                  }
                               }
                            }
                         ]
                      }
                   ]
                }
             }
          ]
       }
    }
`

// BootstrapConfig stores RDS config for Aeraki
type BootstrapConfig struct {
	Address   string
	PortValue string
}

// GetBootstrapConfig gets the ConfigMap to be generated under the namespace
func GetBootstrapConfig(address, portValue string) string {
	bc := BootstrapConfig{
		Address:   address,
		PortValue: portValue,
	}
	var tmplBytes bytes.Buffer
	t := template.Must(template.New("bootstrapConfig").Parse(bootstrapConfig))
	err := t.Execute(&tmplBytes, bc)
	if err != nil {
		log.Errorf("executing template: %v", err)
		os.Exit(1)
	}
	return tmplBytes.String()
}
