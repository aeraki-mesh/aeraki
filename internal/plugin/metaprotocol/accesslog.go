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

package metaprotocol

import (
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v3"
	envoyconfig "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	fileaccesslog "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/file/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"google.golang.org/protobuf/types/known/structpb"
	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pkg/util/protomarshal"
	"istio.io/pkg/log"

	"github.com/aeraki-mesh/aeraki/internal/util/protoconv"
)

const (
	envoyLogFilePath   = "/dev/stdout"
	envoyTextLogFormat = "[%START_TIME%] %REQ(X-META-PROTOCOL-APPLICATION-PROTOCOL)% " +
		"%RESPONSE_CODE% %RESPONSE_CODE_DETAILS% %CONNECTION_TERMINATION_DETAILS% " +
		"\"%UPSTREAM_TRANSPORT_FAILURE_REASON%\" %BYTES_RECEIVED% %BYTES_SENT% " +
		"%DURATION%  \"%REQ(X-FORWARDED-FOR)%\" \"%REQ(X-REQUEST-ID)%\" " +
		"%UPSTREAM_CLUSTER% %UPSTREAM_LOCAL_ADDRESS% %DOWNSTREAM_LOCAL_ADDRESS% " +
		"%DOWNSTREAM_REMOTE_ADDRESS% %ROUTE_NAME%\n"
)

// nolint: lll
var (
	envoyJSONLogFormatIstio = &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"start_time":                        {Kind: &structpb.Value_StringValue{StringValue: "%START_TIME%"}},
			"route_name":                        {Kind: &structpb.Value_StringValue{StringValue: "%ROUTE_NAME%"}},
			"application_protocol":              {Kind: &structpb.Value_StringValue{StringValue: "%REQ(X-META-PROTOCOL-APPLICATION-PROTOCOL)%"}},
			"response_code":                     {Kind: &structpb.Value_StringValue{StringValue: "%RESPONSE_CODE%"}},
			"response_code_details":             {Kind: &structpb.Value_StringValue{StringValue: "%RESPONSE_CODE_DETAILS%"}},
			"connection_termination_details":    {Kind: &structpb.Value_StringValue{StringValue: "%CONNECTION_TERMINATION_DETAILS%"}},
			"bytes_received":                    {Kind: &structpb.Value_StringValue{StringValue: "%BYTES_RECEIVED%"}},
			"bytes_sent":                        {Kind: &structpb.Value_StringValue{StringValue: "%BYTES_SENT%"}},
			"duration":                          {Kind: &structpb.Value_StringValue{StringValue: "%DURATION%"}},
			"request_id":                        {Kind: &structpb.Value_StringValue{StringValue: "%REQ(X-REQUEST-ID)%"}},
			"upstream_cluster":                  {Kind: &structpb.Value_StringValue{StringValue: "%UPSTREAM_CLUSTER%"}},
			"upstream_local_address":            {Kind: &structpb.Value_StringValue{StringValue: "%UPSTREAM_LOCAL_ADDRESS%"}},
			"downstream_local_address":          {Kind: &structpb.Value_StringValue{StringValue: "%DOWNSTREAM_LOCAL_ADDRESS%"}},
			"downstream_remote_address":         {Kind: &structpb.Value_StringValue{StringValue: "%DOWNSTREAM_REMOTE_ADDRESS%"}},
			"upstream_transport_failure_reason": {Kind: &structpb.Value_StringValue{StringValue: "%UPSTREAM_TRANSPORT_FAILURE_REASON%"}},
		},
	}
)

func buildFileAccessLogHelper(path string, mesh *meshconfig.MeshConfig) *accesslog.AccessLog {
	// We need to build access log. This is needed either on first access or when mesh config changes.
	if path == "" {
		path = envoyLogFilePath
	}
	fl := &fileaccesslog.FileAccessLog{
		Path: path,
	}

	switch mesh.AccessLogEncoding {
	case meshconfig.MeshConfig_TEXT:
		formatString := envoyTextLogFormat
		if mesh.AccessLogFormat != "" {
			formatString = mesh.AccessLogFormat
		}
		fl.AccessLogFormat = &fileaccesslog.FileAccessLog_LogFormat{
			LogFormat: &envoyconfig.SubstitutionFormatString{
				Format: &envoyconfig.SubstitutionFormatString_TextFormatSource{
					TextFormatSource: &envoyconfig.DataSource{
						Specifier: &envoyconfig.DataSource_InlineString{
							InlineString: formatString,
						},
					},
				},
			},
		}
	case meshconfig.MeshConfig_JSON:
		jsonLogStruct := envoyJSONLogFormatIstio
		if len(mesh.AccessLogFormat) > 0 {
			parsedJSONLogStruct := structpb.Struct{}
			if err := protomarshal.UnmarshalAllowUnknown([]byte(mesh.AccessLogFormat), &parsedJSONLogStruct); err != nil {
				log.Errorf("error parsing provided json log format, default log format will be used: %v", err)
			} else {
				jsonLogStruct = &parsedJSONLogStruct
			}
		}
		fl.AccessLogFormat = &fileaccesslog.FileAccessLog_LogFormat{
			LogFormat: &envoyconfig.SubstitutionFormatString{
				Format: &envoyconfig.SubstitutionFormatString_JsonFormat{
					JsonFormat: jsonLogStruct,
				},
			},
		}
	default:
		log.Warnf("unsupported access log format %v", mesh.AccessLogEncoding)
	}

	al := &accesslog.AccessLog{
		Name:       wellknown.FileAccessLog,
		ConfigType: &accesslog.AccessLog_TypedConfig{TypedConfig: protoconv.MessageToAny(fl)},
	}

	return al
}
