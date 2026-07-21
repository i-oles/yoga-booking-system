package location

import (
	"fmt"
	"strings"
)

type MemoryResolver struct {
	locations map[string]string
}

func NewMemoryResolver() MemoryResolver {
	locations := map[string]string{
		"ożarowska 75/36":   "https://www.google.com/maps/place/otojoga.art/@52.2460452,20.9469514,608m/data=!3m2!1e3!4b1!4m6!3m5!1s0x471ecbf774653d71:0x642b02d6cc092cd0!8m2!3d52.2460452!4d20.9469514!16s%2Fg%2F11ymv35mkw!5m1!1e1?entry=ttu&g_ep=EgoyMDI2MDYyNC4wIKXMDSoASAFQAw%3D%3D",
		"ogród krasińskich": "https://www.google.com/maps/place/52%C2%B014'56.4%22N+21%C2%B000'08.9%22E/@52.248987,20.9998838,608m/data=!3m2!1e3!4b1!4m4!3m3!8m2!3d52.248987!4d21.0024587!5m1!1e1?entry=ttu&g_ep=EgoyMDI2MDYyNC4wIKXMDSoASAFQAw%3D%3D",
		"ogród saski":       "https://www.google.com/maps?q=52.2413869,21.0059794&entry=gps&lucs=,47071704,94218641&g_ep=CAISDTYuMTA1LjIuNDYzMzAYACDXggMqEiw0NzA3MTcwNCw5NDIxODY0MUICUEw%3D&skid=c43a5e89-5167-446d-b870-2ec468241227&g_st=ia",
		"park moczydło":     "https://www.google.com/maps?q=52.2414050,20.9502750&entry=gps&lucs=,47071704,94218641&g_ep=CAISDTYuMTA1LjIuNDYzMzAYACCenQoqEiw0NzA3MTcwNCw5NDIxODY0MUICUEw%3D&skid=939f844d-13d0-4623-b1c2-64a61e1e3d09&g_st=ia",
	}

	return MemoryResolver{
		locations: locations,
	}
}

func (m MemoryResolver) GetLink(location string) (string, error) {
	n := strings.ToLower(location)

	link, ok := m.locations[n]
	if !ok {
		return "", fmt.Errorf("location link for: %s not found", location)
	}

	return link, nil
}
