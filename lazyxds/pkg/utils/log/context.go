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

package log

import (
	"context"

	"github.com/go-logr/logr"
)

type key int

const (
	logKey key = iota
)

// WithContext returns a copy of parent in which the logger value is set
func WithContext(parent context.Context, logger logr.Logger) context.Context {
	return context.WithValue(parent, logKey, logger)
}

// Ctx returns the logger from context.
func Ctx(ctx context.Context) logr.Logger {
	return FromContext(ctx)
}

// FromContext returns the logger from context.
func FromContext(ctx context.Context) logr.Logger {
	log, ok := ctx.Value(logKey).(logr.Logger)
	if !ok {
		return Logger()
	}

	return log
}
