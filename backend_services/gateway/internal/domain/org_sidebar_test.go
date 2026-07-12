package domain

import (
	"strings"
	"testing"
)

func TestValidateOrgSidebarResponse(t *testing.T) {
	resp := &OrgSidebarResponse{
		AppSlug: "surveillance-pro",
		OrgID:   "org-1",
		Sections: []OrgSidebarSection{
			{
				ID: "main",
				Items: []OrgSidebarItem{
					{ID: "home", Label: "Home", Icon: "home", Path: ""},
					{ID: "users", Label: "Users", Icon: "users", Path: "users"},
					{ID: "organizations", Label: "Organizations", Icon: "layout-grid", Path: "_organizations"},
				},
			},
		},
	}
	if err := ValidateOrgSidebarResponse(resp); err != nil {
		t.Fatalf("expected valid response, got %v", err)
	}
}

func TestValidateOrgSidebarItemInvalidPosition(t *testing.T) {
	resp := &OrgSidebarResponse{
		AppSlug: "surveillance-pro",
		OrgID:   "org-1",
		Sections: []OrgSidebarSection{
			{
				ID: "main",
				Items: []OrgSidebarItem{
					{ID: "settings", Label: "Settings", Icon: "settings", Path: "settings", Position: "middle"},
				},
			},
		},
	}
	if err := ValidateOrgSidebarResponse(resp); err == nil {
		t.Fatal("expected validation error for invalid position")
	}
}

func TestValidateOrgSidebarItemBottomPosition(t *testing.T) {
	resp := &OrgSidebarResponse{
		AppSlug: "surveillance-pro",
		OrgID:   "org-1",
		Sections: []OrgSidebarSection{
			{
				ID: "main",
				Items: []OrgSidebarItem{
					{ID: "home", Label: "Home", Icon: "home", Path: ""},
					{ID: "settings", Label: "Settings", Icon: "settings", Path: "settings", Position: SidebarItemPositionBottom},
				},
			},
		},
	}
	if err := ValidateOrgSidebarResponse(resp); err != nil {
		t.Fatalf("expected valid response with bottom position, got %v", err)
	}
}

func TestValidateOrgSidebarItemLabelTooLong(t *testing.T) {
	longLabel := strings.Repeat("a", SidebarItemLabelMax+1)

	resp := &OrgSidebarResponse{
		AppSlug: "surveillance-pro",
		OrgID:   "org-1",
		Sections: []OrgSidebarSection{
			{
				ID: "main",
				Items: []OrgSidebarItem{
					{ID: "home", Label: longLabel, Icon: "home", Path: ""},
				},
			},
		},
	}
	if err := ValidateOrgSidebarResponse(resp); err == nil {
		t.Fatal("expected validation error for long label")
	}
}
