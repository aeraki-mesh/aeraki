// Copyright 2020 Envoyproxy Authors
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package xds

// An example of a logger that implements `pkg/log/Logger`.  Logs to
// stdout.  If Debug == false then Debugf() and Infof() won't output
// anything.
type logger struct {
}

// Log to stdout only if Debug is true.
func (logger logger) Debugf(format string, args ...interface{}) {
	newArgs := []interface{}{}
	newArgs = append(newArgs, format)
	newArgs = append(newArgs, args...)
	xdsLog.Debugf(newArgs...)
}

// Log to stdout only if Debug is true.
func (logger logger) Infof(format string, args ...interface{}) {
	newArgs := []interface{}{}
	newArgs = append(newArgs, format)
	newArgs = append(newArgs, args...)
	xdsLog.Infof(newArgs...)
}

// Log to stdout always.
func (logger logger) Warnf(format string, args ...interface{}) {
	newArgs := []interface{}{}
	newArgs = append(newArgs, format)
	newArgs = append(newArgs, args...)
	xdsLog.Warnf(newArgs...)
}

// Log to stdout always.
func (logger logger) Errorf(format string, args ...interface{}) {
	newArgs := []interface{}{}
	newArgs = append(newArgs, format)
	newArgs = append(newArgs, args...)
	xdsLog.Errorf(newArgs...)
}
