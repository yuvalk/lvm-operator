package deploymanager

import (
	"context"
	"fmt"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
)

// CreateNamespace creates a namespace in the cluster, ignoring if it already exists.
func (t *DeployManager) CreateNamespace(namespace string) error {
	label := make(map[string]string)
	// Label required for monitoring this namespace
	label["openshift.io/cluster-monitoring"] = "true"
	ns := &k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: label,
		},
	}
	_, err := t.k8sClient.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// DeleteNamespaceAndWait deletes a namespace and waits on it to terminate.
func (t *DeployManager) DeleteNamespaceAndWait(namespace string) error {
	err := t.k8sClient.CoreV1().Namespaces().Delete(context.TODO(), namespace, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	lastReason := ""
	timeout := 600 * time.Second
	interval := 10 * time.Second

	// Wait for namespace to terminate
	err = utilwait.PollImmediate(interval, timeout, func() (done bool, err error) {
		_, err = t.k8sClient.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			lastReason = fmt.Sprintf("Error talking to k8s apiserver: %v", err)
			return false, nil
		}
		if err == nil {
			lastReason = "Waiting on namespace to be deleted"
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return fmt.Errorf("%v: %s", err, lastReason)
	}

	return nil
}
