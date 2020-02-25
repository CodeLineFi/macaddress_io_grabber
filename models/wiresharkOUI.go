package models

import "sort"

type WiresharkOUI struct {
	OUI
	Note string
}

type WiresharkRegistry []*WiresharkOUI

func (r *WiresharkRegistry) Len() int {
	return len(*r)
}

func (r *WiresharkRegistry) Less(i, j int) bool {
	return (*r)[i].Assignment < (*r)[j].Assignment
}

func (r *WiresharkRegistry) Swap(i, j int) {
	(*r)[i], (*r)[j] = (*r)[j], (*r)[i]
}

func (r *WiresharkRegistry) Unique(replace func(old, new *WiresharkOUI) bool) {

	m := map[string]*WiresharkOUI{}

	for _, oui := range *r {
		if old, ok := m[oui.Assignment]; !ok {
			m[oui.Assignment] = oui
		} else if replace != nil && replace(old, oui) {
			m[oui.Assignment] = oui
		}
	}

	var u WiresharkRegistry = make([]*WiresharkOUI, 0, len(m))

	for _, oui := range m {
		u = append(u, oui)
	}

	sort.Sort(&u)

	*r = u
}
