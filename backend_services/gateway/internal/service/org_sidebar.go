package service

import (
	"context"
	"errors"

	"github.com/eternal-orbit-labs/gateway/internal/domain"
)

var ErrSidebarNotConfigured = errors.New("sidebar not configured for app")

type OrgSidebarService struct {
	apps domain.AppRepository
	orgs domain.AppOrgRepository
}

func NewOrgSidebarService(apps domain.AppRepository, orgs domain.AppOrgRepository) *OrgSidebarService {
	return &OrgSidebarService{apps: apps, orgs: orgs}
}

func (s *OrgSidebarService) GetSidebar(ctx context.Context, appSlug, userID, orgID string) (*domain.OrgSidebarResponse, error) {
	appOrgSvc := &AppOrgService{apps: s.apps, orgs: s.orgs}
	if _, err := appOrgSvc.requireAvailableApp(ctx, appSlug); err != nil {
		return nil, err
	}

	org, err := s.orgs.GetOrgForMember(ctx, appSlug, userID, orgID)
	if err != nil {
		return nil, err
	}
	if org == nil {
		return nil, ErrOrgNotFound
	}

	config, ok := sidebarConfigForApp(appSlug)
	if !ok {
		return nil, ErrSidebarNotConfigured
	}

	resp := &domain.OrgSidebarResponse{
		AppSlug:  appSlug,
		OrgID:    orgID,
		Sections: config,
	}
	if err := domain.ValidateOrgSidebarResponse(resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func sidebarConfigForApp(appSlug string) ([]domain.OrgSidebarSection, bool) {
	switch appSlug {
	case "surveillance-pro":
		return []domain.OrgSidebarSection{
			{
				ID: "main",
				Items: []domain.OrgSidebarItem{
					{ID: "home", Label: "Home", Icon: "home", Path: ""},
					{ID: "users", Label: "Users", Icon: "users", Path: "users"},
					{ID: "teams", Label: "Teams", Icon: "user-round", Path: "teams"},
					{ID: "organizations", Label: "Organizations", Icon: "layout-grid", Path: "_organizations"},
					{ID: "settings", Label: "Settings", Icon: "settings", Path: "settings", Position: domain.SidebarItemPositionBottom},
				},
			},
		}, true
	default:
		return nil, false
	}
}
