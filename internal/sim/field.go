package sim

import (
	unitpkg "xqdj/internal/unit"
	_ "xqdj/map"
)

type fieldSpec struct {
	name   string
	shape  string
	extent float64
	hard   []fieldWallSpec
	caps   []fieldCapSpec
}

type fieldWallSpec struct {
	x1, y1, x2, y2, halfW, period float64
}

type fieldCapSpec struct {
	x1, y1, x2, y2, radius, life, amount float64
}

func fieldNames() []string {
	return unitpkg.FieldNames()
}

func lookupField(name string) (fieldSpec, bool) {
	f, ok := unitpkg.LookupField(name)
	if !ok {
		return fieldSpec{}, false
	}
	return specFromField(f), true
}

func defaultField() fieldSpec {
	if spec, ok := lookupField(unitpkg.NameHex); ok {
		return spec
	}
	names := unitpkg.FieldNames()
	if len(names) > 0 {
		if spec, ok := lookupField(names[0]); ok {
			return spec
		}
	}
	return specFromField(unitpkg.HexField())
}

func specFromField(f unitpkg.Field) fieldSpec {
	s := fieldSpec{name: f.Name, shape: f.Shape, extent: f.Extent}
	for _, w := range f.Walls {
		switch {
		case w.Kind.Hard():
			s.hard = append(s.hard, fieldWallSpec{
				x1: w.X1, y1: w.Y1, x2: w.X2, y2: w.Y2, halfW: w.Radius, period: w.Period,
			})
		case w.Kind.Capsule():
			s.caps = append(s.caps, fieldCapSpec{
				x1: w.X1, y1: w.Y1, x2: w.X2, y2: w.Y2, radius: w.Radius,
			})
		}
	}
	return s
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
