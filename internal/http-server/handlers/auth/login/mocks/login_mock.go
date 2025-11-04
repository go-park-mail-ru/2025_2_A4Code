package mocks

import (
	"context"
	"reflect"

	"2025_2_a4code/internal/usecase/profile"

	"github.com/golang/mock/gomock"
)

type MockProfileUsecase struct {
	ctrl     *gomock.Controller
	recorder *MockProfileUsecaseMockRecorder
}

type MockProfileUsecaseMockRecorder struct {
	mock *MockProfileUsecase
}

func NewMockProfileUsecase(ctrl *gomock.Controller) *MockProfileUsecase {
	mock := &MockProfileUsecase{ctrl: ctrl}
	mock.recorder = &MockProfileUsecaseMockRecorder{mock}
	return mock
}

func (m *MockProfileUsecase) EXPECT() *MockProfileUsecaseMockRecorder {
	return m.recorder
}

func (m *MockProfileUsecase) Login(ctx context.Context, req profile.LoginRequest) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Login", ctx, req)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockProfileUsecaseMockRecorder) Login(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Login", reflect.TypeOf((*MockProfileUsecase)(nil).Login), ctx, req)
}

func (m *MockProfileUsecase) Signup(ctx context.Context, req profile.SignupRequest) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Signup", ctx, req)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockProfileUsecaseMockRecorder) Signup(ctx, req interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Signup", reflect.TypeOf((*MockProfileUsecase)(nil).Signup), ctx, req)
}
