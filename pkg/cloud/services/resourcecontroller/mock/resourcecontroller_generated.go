/*
Copyright The Kubernetes Authors.

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
// Code generated by MockGen. DO NOT EDIT.
// Source: ./resourcecontroller.go
//
// Generated by this command:
//
//	mockgen -source=./resourcecontroller.go -destination=./mock/resourcecontroller_generated.go -package=mock
//

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	core "github.com/IBM/go-sdk-core/v5/core"
	resourcecontrollerv2 "github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	gomock "go.uber.org/mock/gomock"
)

// MockResourceController is a mock of ResourceController interface.
type MockResourceController struct {
	ctrl     *gomock.Controller
	recorder *MockResourceControllerMockRecorder
	isgomock struct{}
}

// MockResourceControllerMockRecorder is the mock recorder for MockResourceController.
type MockResourceControllerMockRecorder struct {
	mock *MockResourceController
}

// NewMockResourceController creates a new mock instance.
func NewMockResourceController(ctrl *gomock.Controller) *MockResourceController {
	mock := &MockResourceController{ctrl: ctrl}
	mock.recorder = &MockResourceControllerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockResourceController) EXPECT() *MockResourceControllerMockRecorder {
	return m.recorder
}

// CreateResourceInstance mocks base method.
func (m *MockResourceController) CreateResourceInstance(arg0 *resourcecontrollerv2.CreateResourceInstanceOptions) (*resourcecontrollerv2.ResourceInstance, *core.DetailedResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateResourceInstance", arg0)
	ret0, _ := ret[0].(*resourcecontrollerv2.ResourceInstance)
	ret1, _ := ret[1].(*core.DetailedResponse)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CreateResourceInstance indicates an expected call of CreateResourceInstance.
func (mr *MockResourceControllerMockRecorder) CreateResourceInstance(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateResourceInstance", reflect.TypeOf((*MockResourceController)(nil).CreateResourceInstance), arg0)
}

// CreateResourceKey mocks base method.
func (m *MockResourceController) CreateResourceKey(arg0 *resourcecontrollerv2.CreateResourceKeyOptions) (*resourcecontrollerv2.ResourceKey, *core.DetailedResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateResourceKey", arg0)
	ret0, _ := ret[0].(*resourcecontrollerv2.ResourceKey)
	ret1, _ := ret[1].(*core.DetailedResponse)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CreateResourceKey indicates an expected call of CreateResourceKey.
func (mr *MockResourceControllerMockRecorder) CreateResourceKey(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateResourceKey", reflect.TypeOf((*MockResourceController)(nil).CreateResourceKey), arg0)
}

// DeleteResourceInstance mocks base method.
func (m *MockResourceController) DeleteResourceInstance(arg0 *resourcecontrollerv2.DeleteResourceInstanceOptions) (*core.DetailedResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteResourceInstance", arg0)
	ret0, _ := ret[0].(*core.DetailedResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeleteResourceInstance indicates an expected call of DeleteResourceInstance.
func (mr *MockResourceControllerMockRecorder) DeleteResourceInstance(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteResourceInstance", reflect.TypeOf((*MockResourceController)(nil).DeleteResourceInstance), arg0)
}

// GetInstanceByName mocks base method.
func (m *MockResourceController) GetInstanceByName(arg0, arg1, arg2 string) (*resourcecontrollerv2.ResourceInstance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInstanceByName", arg0, arg1, arg2)
	ret0, _ := ret[0].(*resourcecontrollerv2.ResourceInstance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInstanceByName indicates an expected call of GetInstanceByName.
func (mr *MockResourceControllerMockRecorder) GetInstanceByName(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInstanceByName", reflect.TypeOf((*MockResourceController)(nil).GetInstanceByName), arg0, arg1, arg2)
}

// GetResourceInstance mocks base method.
func (m *MockResourceController) GetResourceInstance(arg0 *resourcecontrollerv2.GetResourceInstanceOptions) (*resourcecontrollerv2.ResourceInstance, *core.DetailedResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResourceInstance", arg0)
	ret0, _ := ret[0].(*resourcecontrollerv2.ResourceInstance)
	ret1, _ := ret[1].(*core.DetailedResponse)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetResourceInstance indicates an expected call of GetResourceInstance.
func (mr *MockResourceControllerMockRecorder) GetResourceInstance(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResourceInstance", reflect.TypeOf((*MockResourceController)(nil).GetResourceInstance), arg0)
}

// GetServiceInstance mocks base method.
func (m *MockResourceController) GetServiceInstance(arg0, arg1 string, arg2 *string) (*resourcecontrollerv2.ResourceInstance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServiceInstance", arg0, arg1, arg2)
	ret0, _ := ret[0].(*resourcecontrollerv2.ResourceInstance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetServiceInstance indicates an expected call of GetServiceInstance.
func (mr *MockResourceControllerMockRecorder) GetServiceInstance(arg0, arg1, arg2 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServiceInstance", reflect.TypeOf((*MockResourceController)(nil).GetServiceInstance), arg0, arg1, arg2)
}

// GetServiceURL mocks base method.
func (m *MockResourceController) GetServiceURL() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServiceURL")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetServiceURL indicates an expected call of GetServiceURL.
func (mr *MockResourceControllerMockRecorder) GetServiceURL() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServiceURL", reflect.TypeOf((*MockResourceController)(nil).GetServiceURL))
}

// ListResourceInstances mocks base method.
func (m *MockResourceController) ListResourceInstances(listResourceInstancesOptions *resourcecontrollerv2.ListResourceInstancesOptions) (*resourcecontrollerv2.ResourceInstancesList, *core.DetailedResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListResourceInstances", listResourceInstancesOptions)
	ret0, _ := ret[0].(*resourcecontrollerv2.ResourceInstancesList)
	ret1, _ := ret[1].(*core.DetailedResponse)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// ListResourceInstances indicates an expected call of ListResourceInstances.
func (mr *MockResourceControllerMockRecorder) ListResourceInstances(listResourceInstancesOptions any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListResourceInstances", reflect.TypeOf((*MockResourceController)(nil).ListResourceInstances), listResourceInstancesOptions)
}

// SetServiceURL mocks base method.
func (m *MockResourceController) SetServiceURL(arg0 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetServiceURL", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetServiceURL indicates an expected call of SetServiceURL.
func (mr *MockResourceControllerMockRecorder) SetServiceURL(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetServiceURL", reflect.TypeOf((*MockResourceController)(nil).SetServiceURL), arg0)
}
