package kube

// nolint:deadcode,golint

import (
	"context"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	metricsutils "github.com/akash-network/node/util/metrics"

	"github.com/akash-network/provider/cluster/kube/builder"
	crd "github.com/akash-network/provider/pkg/apis/akash.network/v2beta2"
	crdapi "github.com/akash-network/provider/pkg/client/clientset/versioned"
)

func applyNS(ctx context.Context, kc kubernetes.Interface, b builder.NS) (*corev1.Namespace, *corev1.Namespace, *corev1.Namespace, error) {
	oobj, err := kc.CoreV1().Namespaces().Get(ctx, b.Name(), metav1.GetOptions{})
	metricsutils.IncCounterVecWithLabelValuesFiltered(kubeCallsCounter, "namespaces-get", err, errors.IsNotFound)

	var nobj *corev1.Namespace
	var uobj *corev1.Namespace

	switch {
	case err == nil:
		curr := oobj.DeepCopy()
		oobj, err = b.Update(oobj)

		if err == nil && (!b.IsObjectRevisionLatest(oobj.Labels) ||
			!reflect.DeepEqual(&curr.Spec, &oobj.Spec) ||
			!reflect.DeepEqual(curr.Labels, oobj.Labels)) {
			uobj, err = kc.CoreV1().Namespaces().Update(ctx, oobj, metav1.UpdateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "namespaces-update", err)
		}
	case errors.IsNotFound(err):
		oobj, err = b.Create()
		if err == nil {
			nobj, err = kc.CoreV1().Namespaces().Create(ctx, oobj, metav1.CreateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "namespaces-create", err)
		}
	}

	return nobj, uobj, oobj, err
}

// Apply list of Network Policies
func applyNetPolicies(ctx context.Context, kc kubernetes.Interface, b builder.NetPol) ([]netv1.NetworkPolicy, []netv1.NetworkPolicy, []netv1.NetworkPolicy, error) {
	var err error

	policies, err := b.Create()
	if err != nil {
		return nil, nil, nil, err
	}

	currPolicies, err := kc.NetworkingV1().NetworkPolicies(b.NS()).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, nil, err
	}

	var nobjs []netv1.NetworkPolicy
	var uobjs []netv1.NetworkPolicy
	oobjs := currPolicies.Items

	for _, pol := range policies {
		oobj, err := kc.NetworkingV1().NetworkPolicies(b.NS()).Get(ctx, pol.Name, metav1.GetOptions{})
		metricsutils.IncCounterVecWithLabelValuesFiltered(kubeCallsCounter, "networking-policies-get", err, errors.IsNotFound)

		var nobj *netv1.NetworkPolicy
		var uobj *netv1.NetworkPolicy

		switch {
		case err == nil:
			curr := oobj.DeepCopy()
			oobj, err = b.Update(oobj)

			if err == nil && (!b.IsObjectRevisionLatest(curr.Labels) ||
				!reflect.DeepEqual(&curr.Spec, &oobj.Spec) ||
				!reflect.DeepEqual(curr.Labels, oobj.Labels)) {
				uobj, err = kc.NetworkingV1().NetworkPolicies(b.NS()).Update(ctx, pol, metav1.UpdateOptions{})
				metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "networking-policies-update", err)
				if err == nil {
					uobjs = append(uobjs, *uobj)
				}
			}
		case errors.IsNotFound(err):
			nobj, err = kc.NetworkingV1().NetworkPolicies(b.NS()).Create(ctx, pol, metav1.CreateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "networking-policies-create", err)
			if err == nil {
				nobjs = append(nobjs, *nobj)
			}
		}

		if err != nil {
			break
		}
	}

	return nobjs, uobjs, oobjs, err
}

func applyServiceCredentials(ctx context.Context, kc kubernetes.Interface, b builder.ServiceCredentials) (*corev1.Secret, *corev1.Secret, *corev1.Secret, error) {
	oobj, err := kc.CoreV1().Secrets(b.NS()).Get(ctx, b.Name(), metav1.GetOptions{})
	metricsutils.IncCounterVecWithLabelValuesFiltered(kubeCallsCounter, "secrets-get", err, errors.IsNotFound)

	var nobj *corev1.Secret
	var uobj *corev1.Secret

	switch {
	case err == nil:
		curr := oobj.DeepCopy()
		oobj, err = b.Update(oobj)
		if err == nil && (!b.IsObjectRevisionLatest(curr.Labels) ||
			!reflect.DeepEqual(&curr.Data, &oobj.Data) ||
			!reflect.DeepEqual(curr.Labels, oobj.Labels)) {
			uobj, err = kc.CoreV1().Secrets(b.NS()).Update(ctx, oobj, metav1.UpdateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "secrets-get", err)
		}
	case errors.IsNotFound(err):
		oobj, err = b.Create()
		if err == nil {
			nobj, err = kc.CoreV1().Secrets(b.NS()).Create(ctx, oobj, metav1.CreateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "secrets-create", err)
		}
	}

	return nobj, uobj, oobj, err

}

func applyDeployment(ctx context.Context, kc kubernetes.Interface, b builder.Deployment) (*appsv1.Deployment, *appsv1.Deployment, *appsv1.Deployment, error) {
	oobj, err := kc.AppsV1().Deployments(b.NS()).Get(ctx, b.Name(), metav1.GetOptions{})
	metricsutils.IncCounterVecWithLabelValuesFiltered(kubeCallsCounter, "deployments-get", err, errors.IsNotFound)

	var nobj *appsv1.Deployment
	var uobj *appsv1.Deployment

	switch {
	case err == nil:
		curr := oobj.DeepCopy()
		oobj, err = b.Update(oobj)
		if err == nil && (b.IsObjectRevisionLatest(curr.Labels) ||
			!reflect.DeepEqual(&curr.Spec, &oobj.Spec) ||
			!reflect.DeepEqual(curr.Labels, oobj.Labels)) {
			uobj, err = kc.AppsV1().Deployments(b.NS()).Update(ctx, oobj, metav1.UpdateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "deployments-update", err)
		}
	case errors.IsNotFound(err):
		oobj, err = b.Create()
		if err == nil {
			nobj, err = kc.AppsV1().Deployments(b.NS()).Create(ctx, oobj, metav1.CreateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "deployments-create", err)
		}
	}

	return nobj, uobj, oobj, err
}

func applyStatefulSet(ctx context.Context, kc kubernetes.Interface, b builder.StatefulSet) (*appsv1.StatefulSet, *appsv1.StatefulSet, *appsv1.StatefulSet, error) {
	oobj, err := kc.AppsV1().StatefulSets(b.NS()).Get(ctx, b.Name(), metav1.GetOptions{})
	metricsutils.IncCounterVecWithLabelValuesFiltered(kubeCallsCounter, "statefulset-get", err, errors.IsNotFound)

	var nobj *appsv1.StatefulSet
	var uobj *appsv1.StatefulSet

	switch {
	case err == nil:
		curr := oobj.DeepCopy()
		oobj, err = b.Update(oobj)
		if err == nil && (b.IsObjectRevisionLatest(curr.Labels) ||
			!reflect.DeepEqual(&curr.Spec, &oobj.Spec) ||
			!reflect.DeepEqual(curr.Labels, oobj.Labels)) {
			uobj, err = kc.AppsV1().StatefulSets(b.NS()).Update(ctx, oobj, metav1.UpdateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "statefulset-update", err)

		}
	case errors.IsNotFound(err):
		oobj, err = b.Create()
		if err == nil {
			nobj, err = kc.AppsV1().StatefulSets(b.NS()).Create(ctx, oobj, metav1.CreateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "statefulset-create", err)
		}
	}

	return nobj, uobj, oobj, err
}

func applyService(ctx context.Context, kc kubernetes.Interface, b builder.Service) (*corev1.Service, *corev1.Service, *corev1.Service, error) {
	oobj, err := kc.CoreV1().Services(b.NS()).Get(ctx, b.Name(), metav1.GetOptions{})
	metricsutils.IncCounterVecWithLabelValuesFiltered(kubeCallsCounter, "services-get", err, errors.IsNotFound)
	var nobj *corev1.Service
	var uobj *corev1.Service

	switch {
	case err == nil:
		curr := oobj.DeepCopy()
		oobj, err = b.Update(oobj)
		if err == nil && (b.IsObjectRevisionLatest(curr.Labels) ||
			!reflect.DeepEqual(&curr.Spec, &oobj.Spec) ||
			!reflect.DeepEqual(curr.Labels, oobj.Labels)) {
			uobj, err = kc.CoreV1().Services(b.NS()).Update(ctx, oobj, metav1.UpdateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "services-update", err)
		}
	case errors.IsNotFound(err):
		oobj, err = b.Create()
		if err == nil {
			nobj, err = kc.CoreV1().Services(b.NS()).Create(ctx, oobj, metav1.CreateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "services-create", err)
		}
	}

	return nobj, uobj, oobj, err
}

func applyManifest(ctx context.Context, kc crdapi.Interface, b builder.Manifest) (*crd.Manifest, *crd.Manifest, *crd.Manifest, error) {
	oobj, err := kc.AkashV2beta2().Manifests(b.NS()).Get(ctx, b.Name(), metav1.GetOptions{})
	metricsutils.IncCounterVecWithLabelValuesFiltered(kubeCallsCounter, "akash-manifests-get", err, errors.IsNotFound)

	var nobj *crd.Manifest
	var uobj *crd.Manifest

	switch {
	case err == nil:
		curr := oobj.DeepCopy()
		oobj, err = b.Update(oobj)
		if err == nil && (!reflect.DeepEqual(&curr.Spec, &oobj.Spec) || !reflect.DeepEqual(curr.Labels, oobj.Labels)) {
			uobj, err = kc.AkashV2beta2().Manifests(b.NS()).Update(ctx, oobj, metav1.UpdateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "akash-manifests-update", err)
		}
	case errors.IsNotFound(err):
		oobj, err = b.Create()
		if err == nil {
			nobj, err = kc.AkashV2beta2().Manifests(b.NS()).Create(ctx, oobj, metav1.CreateOptions{})
			metricsutils.IncCounterVecWithLabelValues(kubeCallsCounter, "akash-manifests-create", err)
		}
	}

	return nobj, uobj, oobj, err
}

// TODO: re-enable.  see #946
// func applyRestrictivePodSecPoliciesToNS(ctx context.Context, kc kubernetes.Interface, p builder.PspRestricted) error {
// 	obj, err := kc.PolicyV1beta1().PodSecurityPolicies().Get(ctx, p.Name(), metav1.GetOptions{})
// 	switch {
// 	case err == nil:
// 		obj, err = p.Update(obj)
// 		if err == nil {
// 			_, err = kc.PolicyV1beta1().PodSecurityPolicies().Update(ctx, obj, metav1.UpdateOptions{})
// 		}
// 	case errors.IsNotFound(err):
// 		obj, err = p.Create()
// 		if err == nil {
// 			_, err = kc.PolicyV1beta1().PodSecurityPolicies().Create(ctx, obj, metav1.CreateOptions{})
// 		}
// 	}
// 	return err
// }
