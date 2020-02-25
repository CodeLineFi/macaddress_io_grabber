package models

import (
	"sort"
)

type OUI struct {
	Assignment string
	Type       string
	OrgName    string
	OrgAddress string
}

type Registry []*OUI

func (r *Registry) Len() int {
	return len(*r)
}

func (r *Registry) Less(i, j int) bool {
	return (*r)[i].Assignment < (*r)[j].Assignment
}

func (r *Registry) Swap(i, j int) {
	(*r)[i], (*r)[j] = (*r)[j], (*r)[i]
}

func (r *Registry) Unique(replace func(old, new *OUI) bool) {

	m := map[string]*OUI{}

	for _, oui := range *r {
		if old, ok := m[oui.Assignment]; !ok {
			m[oui.Assignment] = oui
		} else if replace != nil && replace(old, oui) {
			m[oui.Assignment] = oui
		}
	}

	var u Registry = make([]*OUI, 0, len(m))

	for _, oui := range m {
		u = append(u, oui)
	}

	sort.Sort(&u)

	*r = u
}
