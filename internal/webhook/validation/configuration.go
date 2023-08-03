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

package validation

import (
	"bytes"
	"context"
	"reflect"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

// GenerateWebhookConfig creates ValidationWebhookConfiguration with the Aeraki ca
func GenerateWebhookConfig(caCert *bytes.Buffer, namespace string) error {
	var (
		webhookCfgName = "aeraki-" + namespace
		webhookService = "aeraki"
	)

	kubeClient, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		panic("failed to set go -client")
	}

	path := "/validate"
	fail := admissionregistrationv1.Fail

	sideEffect := admissionregistrationv1.SideEffectClassNone
	config := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookCfgName,
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{{
			Name: "validation.aeraki.io",
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				CABundle: caCert.Bytes(), // CA bundle created earlier
				Service: &admissionregistrationv1.ServiceReference{
					Name:      webhookService,
					Namespace: namespace,
					Path:      &path,
				},
			},
			Rules: []admissionregistrationv1.RuleWithOperations{{Operations: []admissionregistrationv1.OperationType{
				admissionregistrationv1.Create, admissionregistrationv1.Update},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{"metaprotocol.aeraki.io"},
					APIVersions: []string{"*"},
					Resources:   []string{"metarouters"},
				},
			}},
			FailurePolicy:           &fail,
			SideEffects:             &sideEffect,
			AdmissionReviewVersions: []string{"v1", "v1beta1"},
		}},
	}

	old, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(
		context.TODO(), webhookCfgName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			if _, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(
				context.TODO(), config, metav1.CreateOptions{}); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if !reflect.DeepEqual(old.Webhooks, config.Webhooks) {
		config.ResourceVersion = old.ResourceVersion
		if _, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(
			context.TODO(), config, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}
