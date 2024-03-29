// Code generated by MockGen. DO NOT EDIT.
// Source: oauth_app.go

// Package mock is a generated GoMock package.
package mock

import (
	"bkauth/pkg/database/dao"

	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockOAuthAppManager is a mock of OAuthAppManager interface.
type MockOAuthAppManager struct {
	ctrl     *gomock.Controller
	recorder *MockOAuthAppManagerMockRecorder
}

// MockOAuthAppManagerMockRecorder is the mock recorder for MockOAuthAppManager.
type MockOAuthAppManagerMockRecorder struct {
	mock *MockOAuthAppManager
}

// NewMockOAuthAppManager creates a new mock instance.
func NewMockOAuthAppManager(ctrl *gomock.Controller) *MockOAuthAppManager {
	mock := &MockOAuthAppManager{ctrl: ctrl}
	mock.recorder = &MockOAuthAppManagerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOAuthAppManager) EXPECT() *MockOAuthAppManagerMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockOAuthAppManager) Create(app dao.OAuthApp) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", app)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockOAuthAppManagerMockRecorder) Create(app interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockOAuthAppManager)(nil).Create), app)
}

// Exists mocks base method.
func (m *MockOAuthAppManager) Exists(appCode string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Exists", appCode)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Exists indicates an expected call of Exists.
func (mr *MockOAuthAppManagerMockRecorder) Exists(appCode interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Exists", reflect.TypeOf((*MockOAuthAppManager)(nil).Exists), appCode)
}

// Get mocks base method.
func (m *MockOAuthAppManager) Get(appCode string) (dao.OAuthApp, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", appCode)
	ret0, _ := ret[0].(dao.OAuthApp)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockOAuthAppManagerMockRecorder) Get(appCode interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockOAuthAppManager)(nil).Get), appCode)
}

// Update mocks base method.
func (m *MockOAuthAppManager) Update(appCode string, app dao.OAuthApp) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", appCode, app)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockOAuthAppManagerMockRecorder) Update(appCode, app interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockOAuthAppManager)(nil).Update), appCode, app)
}
