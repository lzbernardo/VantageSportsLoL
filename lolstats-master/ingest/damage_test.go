package ingest

import (
	"testing"

	"gopkg.in/tylerb/is.v1"
)

func TestDamageMap(t *testing.T) {
	is := is.New(t)

	d := DamageMap{}
	d.AddDamage(123, 50.0, 50, 5.2)
	d.AddDamage(123, 50.0, 100, 3)
	d.AddDamage(123, 700.0, 500, 10)
	d.AddDeath(123, 700)
	d.AddDamage(123, 12345678, 100, 5.2)
	d.AddDamage(95, 12345678, 1000, 15.1)

	is.Equal(8.2, d["123"]["early"].DamagePercent)
	is.Equal(150, d["123"]["early"].DamageTotal)
	is.Equal(0, d["123"]["early"].Deaths)

	is.Equal(10, d["123"]["mid"].DamagePercent)
	is.Equal(500, d["123"]["mid"].DamageTotal)
	is.Equal(1, d["123"]["mid"].Deaths)

	is.Equal(5.2, d["123"]["late"].DamagePercent)
	is.Equal(100, d["123"]["late"].DamageTotal)
	is.Equal(0, d["123"]["late"].Deaths)

	is.Equal(15.1, d["95"]["late"].DamagePercent)
	is.Equal(1000, d["95"]["late"].DamageTotal)
	is.Equal(0, d["95"]["late"].Deaths)
}
