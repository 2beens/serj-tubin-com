// Code generated by MockGen. DO NOT EDIT.
// Source: tracker.go
//
// Generated by this command:
//
//	mockgen -source=tracker.go -destination=tracker_mocks_test.go -package=spotify_test
//

// Package spotify_test is a generated GoMock package.
package spotify_test

import (
	context "context"
	reflect "reflect"
	time "time"

	spotify "github.com/2beens/serjtubincom/internal/spotify"
	spotify0 "github.com/zmb3/spotify/v2"
	gomock "go.uber.org/mock/gomock"
)

// MocktracksRepo is a mock of tracksRepo interface.
type MocktracksRepo struct {
	ctrl     *gomock.Controller
	recorder *MocktracksRepoMockRecorder
}

// MocktracksRepoMockRecorder is the mock recorder for MocktracksRepo.
type MocktracksRepoMockRecorder struct {
	mock *MocktracksRepo
}

// NewMocktracksRepo creates a new mock instance.
func NewMocktracksRepo(ctrl *gomock.Controller) *MocktracksRepo {
	mock := &MocktracksRepo{ctrl: ctrl}
	mock.recorder = &MocktracksRepoMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MocktracksRepo) EXPECT() *MocktracksRepoMockRecorder {
	return m.recorder
}

// Add mocks base method.
func (m *MocktracksRepo) Add(arg0 context.Context, arg1 spotify.TrackDBRecord) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Add", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Add indicates an expected call of Add.
func (mr *MocktracksRepoMockRecorder) Add(arg0, arg1 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*MocktracksRepo)(nil).Add), arg0, arg1)
}

// GetLastPlayedTrackTime mocks base method.
func (m *MocktracksRepo) GetLastPlayedTrackTime(arg0 context.Context) (time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetLastPlayedTrackTime", arg0)
	ret0, _ := ret[0].(time.Time)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLastPlayedTrackTime indicates an expected call of GetLastPlayedTrackTime.
func (mr *MocktracksRepoMockRecorder) GetLastPlayedTrackTime(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLastPlayedTrackTime", reflect.TypeOf((*MocktracksRepo)(nil).GetLastPlayedTrackTime), arg0)
}

// MockspotifyClient is a mock of spotifyClient interface.
type MockspotifyClient struct {
	ctrl     *gomock.Controller
	recorder *MockspotifyClientMockRecorder
}

// MockspotifyClientMockRecorder is the mock recorder for MockspotifyClient.
type MockspotifyClientMockRecorder struct {
	mock *MockspotifyClient
}

// NewMockspotifyClient creates a new mock instance.
func NewMockspotifyClient(ctrl *gomock.Controller) *MockspotifyClient {
	mock := &MockspotifyClient{ctrl: ctrl}
	mock.recorder = &MockspotifyClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockspotifyClient) EXPECT() *MockspotifyClientMockRecorder {
	return m.recorder
}

// PlayerRecentlyPlayedOpt mocks base method.
func (m *MockspotifyClient) PlayerRecentlyPlayedOpt(ctx context.Context, opt *spotify0.RecentlyPlayedOptions) ([]spotify0.RecentlyPlayedItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PlayerRecentlyPlayedOpt", ctx, opt)
	ret0, _ := ret[0].([]spotify0.RecentlyPlayedItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// PlayerRecentlyPlayedOpt indicates an expected call of PlayerRecentlyPlayedOpt.
func (mr *MockspotifyClientMockRecorder) PlayerRecentlyPlayedOpt(ctx, opt any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PlayerRecentlyPlayedOpt", reflect.TypeOf((*MockspotifyClient)(nil).PlayerRecentlyPlayedOpt), ctx, opt)
}