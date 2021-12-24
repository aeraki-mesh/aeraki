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

package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (c *AggregationController) updateDiscoverySelector(discoverySelector []*metav1.LabelSelector) error {
	var selectors []labels.Selector
	// convert LabelSelectors to Selectors
	for _, selector := range discoverySelector {
		ls, err := metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			c.log.Error(err, "error initializing discovery namespaces filter, invalid discovery selector: %v")
			return err
		}
		selectors = append(selectors, ls)
	}

	c.selectors = selectors
	return nil
}
