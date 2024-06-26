// Code generated by MockGen. DO NOT EDIT.
// Source: exercises_handler.go

// Package exercises_test is a generated GoMock package.
package exercises_test

import (
	context "context"
	reflect "reflect"

	exercises "github.com/2beens/serjtubincom/internal/gymstats/exercises"
	gomock "github.com/golang/mock/gomock"
)

// MockexercisesRepo is a mock of exercisesRepo interface.
type MockexercisesRepo struct {
	ctrl     *gomock.Controller
	recorder *MockexercisesRepoMockRecorder
}

// MockexercisesRepoMockRecorder is the mock recorder for MockexercisesRepo.
type MockexercisesRepoMockRecorder struct {
	mock *MockexercisesRepo
}

// NewMockexercisesRepo creates a new mock instance.
func NewMockexercisesRepo(ctrl *gomock.Controller) *MockexercisesRepo {
	mock := &MockexercisesRepo{ctrl: ctrl}
	mock.recorder = &MockexercisesRepoMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockexercisesRepo) EXPECT() *MockexercisesRepoMockRecorder {
	return m.recorder
}

// Add mocks base method.
func (m *MockexercisesRepo) Add(ctx context.Context, exercise exercises.Exercise) (*exercises.Exercise, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Add", ctx, exercise)
	ret0, _ := ret[0].(*exercises.Exercise)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Add indicates an expected call of Add.
func (mr *MockexercisesRepoMockRecorder) Add(ctx, exercise interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*MockexercisesRepo)(nil).Add), ctx, exercise)
}

// Delete mocks base method.
func (m *MockexercisesRepo) Delete(ctx context.Context, id int) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockexercisesRepoMockRecorder) Delete(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockexercisesRepo)(nil).Delete), ctx, id)
}

// ExercisesCount mocks base method.
func (m *MockexercisesRepo) ExercisesCount(ctx context.Context, params exercises.ListParams) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExercisesCount", ctx, params)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExercisesCount indicates an expected call of ExercisesCount.
func (mr *MockexercisesRepoMockRecorder) ExercisesCount(ctx, params interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExercisesCount", reflect.TypeOf((*MockexercisesRepo)(nil).ExercisesCount), ctx, params)
}

// Get mocks base method.
func (m *MockexercisesRepo) Get(ctx context.Context, id int) (*exercises.Exercise, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, id)
	ret0, _ := ret[0].(*exercises.Exercise)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockexercisesRepoMockRecorder) Get(ctx, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockexercisesRepo)(nil).Get), ctx, id)
}

// GetExerciseTypes mocks base method.
func (m *MockexercisesRepo) GetExerciseTypes(ctx context.Context, params exercises.GetExerciseTypesParams) ([]exercises.ExerciseType, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExerciseTypes", ctx, params)
	ret0, _ := ret[0].([]exercises.ExerciseType)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetExerciseTypes indicates an expected call of GetExerciseTypes.
func (mr *MockexercisesRepoMockRecorder) GetExerciseTypes(ctx, params interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExerciseTypes", reflect.TypeOf((*MockexercisesRepo)(nil).GetExerciseTypes), ctx, params)
}

// List mocks base method.
func (m *MockexercisesRepo) List(ctx context.Context, params exercises.ListParams) ([]exercises.Exercise, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, params)
	ret0, _ := ret[0].([]exercises.Exercise)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// List indicates an expected call of List.
func (mr *MockexercisesRepoMockRecorder) List(ctx, params interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockexercisesRepo)(nil).List), ctx, params)
}

// ListAll mocks base method.
func (m *MockexercisesRepo) ListAll(ctx context.Context, params exercises.ExerciseParams) ([]exercises.Exercise, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ListAll", ctx, params)
	ret0, _ := ret[0].([]exercises.Exercise)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListAll indicates an expected call of ListAll.
func (mr *MockexercisesRepoMockRecorder) ListAll(ctx, params interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListAll", reflect.TypeOf((*MockexercisesRepo)(nil).ListAll), ctx, params)
}

// Update mocks base method.
func (m *MockexercisesRepo) Update(ctx context.Context, exercise *exercises.Exercise) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, exercise)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockexercisesRepoMockRecorder) Update(ctx, exercise interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockexercisesRepo)(nil).Update), ctx, exercise)
}
