/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta2

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-infrastructure-cluster-x-k8s-io-v1beta2-ibmvpcmachine,mutating=true,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=ibmvpcmachines,verbs=create;update,versions=v1beta2,name=mibmvpcmachine.kb.io,sideEffects=None,admissionReviewVersions=v1;v1beta1
//+kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1beta2-ibmvpcmachine,mutating=false,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=ibmvpcmachines,versions=v1beta2,name=vibmvpcmachine.kb.io,sideEffects=None,admissionReviewVersions=v1;v1beta1

func (r *IBMVPCMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&IBMVPCMachine{}).
		WithValidator(r).
		WithDefaulter(r).
		Complete()
}

var _ webhook.CustomDefaulter = &IBMVPCMachine{}
var _ webhook.CustomValidator = &IBMVPCMachine{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type.
func (r *IBMVPCMachine) Default(_ context.Context, obj runtime.Object) error {
	objValue, ok := obj.(*IBMVPCMachine)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a IBMVPCMachine but got a %T", obj))
	}
	defaultIBMVPCMachineSpec(&objValue.Spec)
	return nil
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type.
func (r *IBMVPCMachine) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	objValue, ok := obj.(*IBMVPCMachine)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a IBMVPCMachine but got a %T", obj))
	}
	var allErrs field.ErrorList
	allErrs = append(allErrs, objValue.validateIBMVPCMachineBootVolume()...)
	return nil, aggregateObjErrors(objValue.GroupVersionKind().GroupKind(), objValue.Name, allErrs)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type.
func (r *IBMVPCMachine) ValidateUpdate(_ context.Context, _, _ runtime.Object) (warnings admission.Warnings, err error) {
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type.
func (r *IBMVPCMachine) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (r *IBMVPCMachine) validateIBMVPCMachineBootVolume() field.ErrorList {
	return validateBootVolume(r.Spec)
}
