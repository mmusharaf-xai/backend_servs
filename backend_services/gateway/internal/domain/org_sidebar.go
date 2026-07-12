package domain

import (
	"fmt"
	"strings"
)

const (
	SidebarItemLabelMax    = 32
	SidebarSectionLabelMax = 24
	SidebarIDMax           = 64

	SidebarItemPositionTop    = "top"
	SidebarItemPositionBottom = "bottom"
)

// OrgSidebarItem is a single navigable entry in an org workspace sidebar.
type OrgSidebarItem struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Icon     string `json:"icon"`
	Path     string `json:"path"`
	Position string `json:"position,omitempty"` // top (default) | bottom
}

// OrgSidebarSection groups sidebar items; omit Label for ungrouped flat items.
type OrgSidebarSection struct {
	ID              string           `json:"id"`
	Label           *string          `json:"label,omitempty"`
	Collapsible     bool             `json:"collapsible,omitempty"`
	DefaultExpanded bool             `json:"default_expanded,omitempty"`
	Items           []OrgSidebarItem `json:"items"`
}

// OrgSidebarResponse is the sidebar configuration for an org workspace.
type OrgSidebarResponse struct {
	AppSlug  string              `json:"app_slug"`
	OrgID    string              `json:"org_id"`
	Sections []OrgSidebarSection `json:"sections"`
}

func ValidateOrgSidebarResponse(resp *OrgSidebarResponse) error {
	if resp == nil {
		return fmt.Errorf("sidebar response is nil")
	}
	if strings.TrimSpace(resp.AppSlug) == "" {
		return fmt.Errorf("app_slug is required")
	}
	if strings.TrimSpace(resp.OrgID) == "" {
		return fmt.Errorf("org_id is required")
	}
	for _, section := range resp.Sections {
		if err := validateOrgSidebarSection(section); err != nil {
			return err
		}
	}
	return nil
}

func validateOrgSidebarSection(section OrgSidebarSection) error {
	if strings.TrimSpace(section.ID) == "" {
		return fmt.Errorf("section id is required")
	}
	if len(section.ID) > SidebarIDMax {
		return fmt.Errorf("section id must be at most %d characters", SidebarIDMax)
	}
	if section.Label != nil {
		label := strings.TrimSpace(*section.Label)
		if label == "" {
			return fmt.Errorf("section label must not be empty when provided")
		}
		if len(label) > SidebarSectionLabelMax {
			return fmt.Errorf("section label must be at most %d characters", SidebarSectionLabelMax)
		}
	}
	if len(section.Items) == 0 {
		return fmt.Errorf("section %q must have at least one item", section.ID)
	}
	for _, item := range section.Items {
		if err := validateOrgSidebarItem(item); err != nil {
			return err
		}
	}
	return nil
}

func validateOrgSidebarItem(item OrgSidebarItem) error {
	if strings.TrimSpace(item.ID) == "" {
		return fmt.Errorf("item id is required")
	}
	if len(item.ID) > SidebarIDMax {
		return fmt.Errorf("item id must be at most %d characters", SidebarIDMax)
	}
	label := strings.TrimSpace(item.Label)
	if label == "" {
		return fmt.Errorf("item label is required")
	}
	if len(label) > SidebarItemLabelMax {
		return fmt.Errorf("item label must be at most %d characters", SidebarItemLabelMax)
	}
	if strings.TrimSpace(item.Icon) == "" {
		return fmt.Errorf("item icon is required")
	}
	if strings.Contains(item.Path, "/") {
		return fmt.Errorf("item path must not contain slashes")
	}
	position := strings.TrimSpace(item.Position)
	if position == "" {
		position = SidebarItemPositionTop
	}
	if position != SidebarItemPositionTop && position != SidebarItemPositionBottom {
		return fmt.Errorf("item position must be top or bottom")
	}
	return nil
}
