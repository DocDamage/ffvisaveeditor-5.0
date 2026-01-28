package config

type MapPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func GetMapPoint(world int, name string) (MapPoint, bool) {
	if data.WorldMapPoints == nil {
		return MapPoint{}, false
	}
	pointsByName, ok := data.WorldMapPoints[world]
	if !ok || pointsByName == nil {
		return MapPoint{}, false
	}
	p, ok := pointsByName[name]
	return p, ok
}

func AllMapPoints(world int) map[string]MapPoint {
	if data.WorldMapPoints == nil {
		return map[string]MapPoint{}
	}
	pointsByName, ok := data.WorldMapPoints[world]
	if !ok || pointsByName == nil {
		return map[string]MapPoint{}
	}
	cp := make(map[string]MapPoint, len(pointsByName))
	for k, v := range pointsByName {
		cp[k] = v
	}
	return cp
}

func SetMapPoint(world int, name string, p MapPoint) {
	if data.WorldMapPoints == nil {
		data.WorldMapPoints = map[int]map[string]MapPoint{}
	}
	if data.WorldMapPoints[world] == nil {
		data.WorldMapPoints[world] = map[string]MapPoint{}
	}
	data.WorldMapPoints[world][name] = p
	save()
}

func ClearMapPoint(world int, name string) {
	if data.WorldMapPoints == nil {
		return
	}
	if data.WorldMapPoints[world] == nil {
		return
	}
	delete(data.WorldMapPoints[world], name)
	save()
}

