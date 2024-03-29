// Code generated by MockGen. DO NOT EDIT.
// Source: access_key.go

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	"bkauth/pkg/service/types"

	gomock "github.com/golang/mock/gomock"
)

// MockAccessKeyService is a mock of AccessKeyService interface.
type MockAccessKeyService struct {
	ctrl     *gomock.Controller
	recorder *MockAccessKeyServiceMockRecorder
}

// MockAccessKeyServiceMockRecorder is the mock recorder for MockAccessKeyService.
type MockAccessKeyServiceMockRecorder struct {
	mock *MockAccessKeyService
}

// NewMockAccessKeyService creates a new mock instance.
func NewMockAccessKeyService(ctrl *gomock.Controller) *MockAccessKeyService {
	mock := &MockAccessKeyService{ctrl: ctrl}
	mock.recorder = &MockAccessKeyServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAccessKeyService) EXPECT() *MockAccessKeyServiceMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockAccessKeyService) Create(appCode, createdSource string) (types.AccessKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", appCode, createdSource)
	ret0, _ := ret[0].(types.AccessKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockAccessKeyServiceMockRecorder) Create(appCode, createdSource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockAccessKeyService)(nil).Create), appCode, createdSource)
}

// CreateWithSecret mocks base method.
func (m *MockAccessKeyService) CreateWithSecret(appCode, appSecret, createdSource string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateWithSecret", appCode, appSecret, createdSource)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateWithSecret indicates an expected call of CreateWithSecret.
func (mr *MockAccessKeyServiceMockRecorder) CreateWithSecret(appCode, appSecret, createdSource interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateWithSecret", reflect.TypeOf((*MockAccessKeyService)(nil).CreateWithSecret), appCode, appSecret, createdSource)
}

// DeleteByID mocks base method.
func (m *MockAccessKeyService) DeleteByID(appCode string, id int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteByID", appCode, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteByID indicates an expected call of DeleteByID.
func (mr *MockAccessKeyServiceMockRecorder) DeleteByID(appCode, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteByID", reflect.TypeOf((*MockAccessKeyService)(nil).DeleteByID), appCode, id)
}

// List mocks base method.
func (m *MockAccessKeyService) List() ([]types.AccessKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List")
	ret0, _ := ret[0].([]types.AccessKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockAccessKeyServiceMockRecorder) List() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockAccessKeyService)(nil).List))
}

// ListEncryptedAccessKeyByAppCode mocks base method.
func (m *MockAccessKeyService) ListEncryptedAccessKeyByAppCode(appCode string) ([]types.AccessKey, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListEncryptedAccessKeyByAppCode", appCode)
	ret0, _ := ret[0].([]types.AccessKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListEncryptedAccessKeyByAppCode indicates an expected call of ListEncryptedAccessKeyByAppCode.
func (mr *MockAccessKeyServiceMockRecorder) ListEncryptedAccessKeyByAppCode(appCode interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListEncryptedAccessKeyByAppCode", reflect.TypeOf((*MockAccessKeyService)(nil).ListEncryptedAccessKeyByAppCode), appCode)
}

// ListWithCreatedAtByAppCode mocks base method.
func (m *MockAccessKeyService) ListWithCreatedAtByAppCode(appCode string) ([]types.AccessKeyWithCreatedAt, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListWithCreatedAtByAppCode", appCode)
	ret0, _ := ret[0].([]types.AccessKeyWithCreatedAt)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListWithCreatedAtByAppCode indicates an expected call of ListWithCreatedAtByAppCode.
func (mr *MockAccessKeyServiceMockRecorder) ListWithCreatedAtByAppCode(appCode interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListWithCreatedAtByAppCode", reflect.TypeOf((*MockAccessKeyService)(nil).ListWithCreatedAtByAppCode), appCode)
}

// UpdateByID mocks base method.
func (m *MockAccessKeyService) UpdateByID(id int64, updateFiledMap map[string]interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateByID", id, updateFiledMap)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateByID indicates an expected call of UpdateByID.
func (mr *MockAccessKeyServiceMockRecorder) UpdateByID(id, updateFiledMap interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateByID", reflect.TypeOf((*MockAccessKeyService)(nil).UpdateByID), id, updateFiledMap)
}

// Verify mocks base method.
func (m *MockAccessKeyService) Verify(appCode, appSecret string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Verify", appCode, appSecret)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Verify indicates an expected call of Verify.
func (mr *MockAccessKeyServiceMockRecorder) Verify(appCode, appSecret interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Verify", reflect.TypeOf((*MockAccessKeyService)(nil).Verify), appCode, appSecret)
}
