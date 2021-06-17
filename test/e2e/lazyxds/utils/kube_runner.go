/*
 * // Copyright Aeraki Authors
 * //
 * // Licensed under the Apache License, Version 2.0 (the "License");
 * // you may not use this file except in compliance with the License.
 * // You may obtain a copy of the License at
 * //
 * //     http://www.apache.org/licenses/LICENSE-2.0
 * //
 * // Unless required by applicable law or agreed to in writing, software
 * // distributed under the License is distributed on an "AS IS" BASIS,
 * // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * // See the License for the specific language governing permissions and
 * // limitations under the License.
 */

package utils

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"log"
	"os"
	"strings"
	"time"
)

// KubeRunner represent the command running in the container
type KubeRunner struct {
	client *kubernetes.Clientset
	config *rest.Config
}

// NewKubeRunnerFromENV ...
func NewKubeRunnerFromENV() (*KubeRunner, error) {
	runner := &KubeRunner{}

	kubeConfFile := os.Getenv("KUBECONFIG")
	if kubeConfFile == "" {
		dirname, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		kubeConfFile = fmt.Sprintf("%s/.kube/config", dirname) // todo check exists
		//return nil, fmt.Errorf("miss env KUBECONFIG")
	}

	if kubeConf, err := clientcmd.BuildConfigFromFlags("", kubeConfFile); err == nil {
		runner.config = kubeConf
	} else {
		return nil, fmt.Errorf("build kube client config from config file failed: %w", err)
	}

	if kubeClient, err := kubernetes.NewForConfig(runner.config); err == nil {
		runner.client = kubeClient
	} else {
		return nil, fmt.Errorf("create kube client failed:  %w", err)
	}

	return runner, nil
}

// ExecPod execute the command in the specified pod
func (r *KubeRunner) ExecPod(container, podName, namespace, command string) (string, error) {
	log.Printf("pod %s exce cmd: %s\n", podName, command)
	req := r.client.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(namespace).SubResource("exec")
	option := &v1.PodExecOptions{
		Container: container,
		Command:   []string{"sh", "-c", command}, // strings.Fields(command),
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)

	var stdout, stderr bytes.Buffer
	executor, err := remotecommand.NewSPDYExecutor(r.config, "POST", req.URL())
	if err != nil {
		log.Printf("NewSPDYExecutor error: %v\n", err)
		return "", err
	}
	err = executor.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    true,
	})
	if err != nil {
		log.Printf("exce stdout: %s\n", stderr.String())
		return "", err
	}

	return strings.TrimSpace(stdout.String()), err
}

// XDSStatistics get the cds and eds statistics of pod
func (r *KubeRunner) XDSStatistics(name, ns string) string {
	return fmt.Sprintf("XDS statistics of %s/%s: CDS count: %d EDS count: %d",
		ns, name,
		r.CountOfCDS(name, ns),
		r.CountOfEDS(name, ns))
}

// CountOfCDS get cds count of pod
func (r *KubeRunner) CountOfCDS(name, ns string) int {
	s := RunCMD(fmt.Sprintf("istioctl pc cluster -n %s %s", ns, name))
	cds := strings.Split(s, "\n")

	return len(cds) - 1 // remove header
}

// CountOfEDS get eds count of pod
func (r *KubeRunner) CountOfEDS(name, ns string) int {
	s := RunCMD(fmt.Sprintf("istioctl pc endpoints -n %s %s", ns, name))
	cds := strings.Split(s, "\n")

	return len(cds) - 1 // remove header
}

// GetAccessLog get the access log of container
func (r *KubeRunner) GetAccessLog(container, podName, ns string, since time.Time, match string) (string, error) {
	rs, err := r.client.CoreV1().Pods(ns).GetLogs(podName, &v1.PodLogOptions{
		Follow:    true,
		SinceTime: &metav1.Time{Time: since},
		Container: container,
	}).Stream(context.TODO())

	if err != nil {
		return "", err
	}
	defer rs.Close()

	// todo may need timeout control
	sc := bufio.NewScanner(rs)

	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(line, match) {
			return line, nil
		}
	}
	return "", fmt.Errorf("log not found")
}

// GetServiceIP get service ip by service name and namespace
func (r *KubeRunner) GetServiceIP(name, ns string) string {
	svc, _ := r.client.CoreV1().Services(ns).Get(context.TODO(), name, metav1.GetOptions{})

	return svc.Spec.ClusterIP
}

// GetFirstPodByLabels get the first pod by labels
func (r *KubeRunner) GetFirstPodByLabels(namespace, labels string) (*corev1.Pod, error) {
	pods, err := r.client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels,
	})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pod found with label %s", labels)
	}
	return &pods.Items[0], nil
}

// CreateNamespace creates namespace, with the control of istio-injection label
func (r *KubeRunner) CreateNamespace(namespace string, inject bool) error {
	ns := &v1.Namespace{
		ObjectMeta: v12.ObjectMeta{
			Name:   namespace,
			Labels: map[string]string{},
		},
	}

	if inject {
		// ns.Labels["istio.io/rev"] = "1-8-1"
		ns.Labels["istio-injection"] = "enabled"
	}

	_, err := r.client.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	return err
}
