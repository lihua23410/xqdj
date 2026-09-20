package sim

import unitpkg "xqdj/internal/unit"

type fieldSpec struct {
	name   string
	shape  string
	extent float64
	hard   []fieldWallSpec
	caps   []fieldCapSpec
}

type fieldWallSpec struct {
	x1, y1, x2, y2, halfW float64
}

type fieldCapSpec struct {
	x1, y1, x2, y2, radius, life, amount float64
}

func fieldCatalog() []fieldSpec {
	return []fieldSpec{
		{name: unitpkg.NameHex, shape: unitpkg.ShapeHex, extent: HexRadius},
		{
			name:   unitpkg.NameCircle,
			shape:  unitpkg.ShapeCircle,
			extent: HexRadius,
			hard:   []fieldWallSpec{{x1: -110, y1: 0, x2: 110, y2: 0, halfW: 6}},
		},
	}
}

func fieldNames() []string {
	cat := fieldCatalog()
	out := make([]string, len(cat))
	for i, f := range cat {
		out[i] = f.name
	}
	return out
}

func fieldByName(name string) fieldSpec {
	for _, f := range fieldCatalog() {
		if f.name == name {
			return f
		}
	}
	return fieldCatalog()[0]
}

func (s fieldSpec) hex() hexagon {
	return newHexagon(s.extent)
}

func (s fieldSpec) toUnitField() unitpkg.Field {
	f := unitpkg.Field{Name: s.name, Shape: s.shape, Extent: s.extent}
	for _, h := range s.hard {
		f.Walls = append(f.Walls, unitpkg.FieldWall{
			Kind: unitpkg.WallHard,
			X1:   h.x1, Y1: h.y1, X2: h.x2, Y2: h.y2,
			Radius: h.halfW,
		})
	}
	for _, c := range s.caps {
		f.Walls = append(f.Walls, unitpkg.FieldWall{
			Kind: unitpkg.WallCapsule,
			X1:   c.x1, Y1: c.y1, X2: c.x2, Y2: c.y2,
			Radius: c.radius,
		})
	}
	return f
}
