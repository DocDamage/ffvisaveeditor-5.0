package config

import "strings"

func MapLocations(world int) []string {
	if data.WorldMapLocations == nil {
		return []string{}
	}
	locs := data.WorldMapLocations[world]
	if locs == nil {
		return []string{}
	}
	out := make([]string, 0, len(locs))
	seen := map[string]bool{}
	for _, v := range locs {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, v)
	}
	return out
}

func AddMapLocation(world int, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if data.WorldMapLocations == nil {
		data.WorldMapLocations = map[int][]string{}
	}
	cur := data.WorldMapLocations[world]
	key := strings.ToLower(name)
	for _, v := range cur {
		if strings.ToLower(strings.TrimSpace(v)) == key {
			return
		}
	}
	data.WorldMapLocations[world] = append(cur, name)
	save()
}

func RemoveMapLocation(world int, name string) {
	if data.WorldMapLocations == nil {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	cur := data.WorldMapLocations[world]
	if len(cur) == 0 {
		return
	}
	key := strings.ToLower(name)
	out := cur[:0]
	for _, v := range cur {
		if strings.ToLower(strings.TrimSpace(v)) == key {
			continue
		}
		out = append(out, v)
	}
	data.WorldMapLocations[world] = out
	save()
}

