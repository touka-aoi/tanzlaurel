package domain_test

import (
	"testing"

	domain "withered/server/domain"
	"withered/server/domain/mocks"
	"go.uber.org/mock/gomock"
)

// 初期化時にリソースが正しくセットアップされることを確認
func TestNewSessionEndpoint_InitializesDefaults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	s := domain.NewSession()
	tr := mocks.NewMockTransport(ctrl)
	c := domain.NewConnection(s.ID(), tr)
	ps := mocks.NewMockPubSub(ctrl)
	rm := mocks.NewMockRoomManager(ctrl)

	se, err := domain.NewSessionEndpoint(s, c, ps, rm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if se == nil {
		t.Fatalf("endpoint is nil")
	}
}
