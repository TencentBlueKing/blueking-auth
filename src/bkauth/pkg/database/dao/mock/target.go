// Code generated by MockGen. DO NOT EDIT.
// Source: target.go
//
// Generated by this command:
//
//	mockgen -source=target.go -destination=./mock/target.go -package=mock
//

// Package mock is a generated GoMock package.
package mock

import (
	dao "bkauth/pkg/database/dao"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockTargetManager is a mock of TargetManager interface.
type MockTargetManager struct {
	ctrl     *gomock.Controller
	recorder *MockTargetManagerMockRecorder
	isgomock struct{}
}

// MockTargetManagerMockRecorder is the mock recorder for MockTargetManager.
type MockTargetManagerMockRecorder struct {
	mock *MockTargetManager
}

// NewMockTargetManager creates a new mock instance.
func NewMockTargetManager(ctrl *gomock.Controller) *MockTargetManager {
	mock := &MockTargetManager{ctrl: ctrl}
	mock.recorder = &MockTargetManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTargetManager) EXPECT() *MockTargetManagerMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockTargetManager) Create(target dao.Target) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", target)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockTargetManagerMockRecorder) Create(target any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockTargetManager)(nil).Create), target)
}

// Exists mocks base method.
func (m *MockTargetManager) Exists(id string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exists", id)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exists indicates an expected call of Exists.
func (mr *MockTargetManagerMockRecorder) Exists(id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exists", reflect.TypeOf((*MockTargetManager)(nil).Exists), id)
}

// Get mocks base method.
func (m *MockTargetManager) Get(id string) (dao.Target, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", id)
	ret0, _ := ret[0].(dao.Target)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockTargetManagerMockRecorder) Get(id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockTargetManager)(nil).Get), id)
}

// Update mocks base method.
func (m *MockTargetManager) Update(id string, target dao.Target) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", id, target)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockTargetManagerMockRecorder) Update(id, target any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockTargetManager)(nil).Update), id, target)
}
