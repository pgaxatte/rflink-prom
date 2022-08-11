package main

import "testing"

func TestParseTemp_ok(t *testing.T) {
	var (
		f   float64
		err error
	)

	inputs := [...]string{"010A", "7FFF", "0000", "810A", "FFFF", "8000"}
	expects := [...]float64{26.6, 3276.7, 0.0, -26.6, -3276.7, 0.0}

	for i := 0; i < len(inputs); i++ {
		f, err = parseTemp(inputs[i])
		if err != nil {
			t.Logf("An error was not expected for value %s: %s", inputs[i], err)
			t.Fail()
		}
		if f != expects[i] {
			t.Logf("Expected %f but got %f", expects[i], f)
			t.Fail()
		}
	}
}

func TestParseTemp_errors(t *testing.T) {
	var err error

	inputs := [...]string{"nope", "1AB", "FFFFF"}

	for i := 0; i < len(inputs); i++ {
		_, err = parseTemp(inputs[i])
		if err == nil {
			t.Logf("An error was expected for value %s", inputs[i])
			t.Fail()
		}
	}
}

func TestStrToFloat_ok(t *testing.T) {
	var (
		f   float64
		err error
	)

	inputs := [...]string{"010A", "7FFF", "0000", "1024"}
	bases := [...]int{16, 16, 16, 10}
	expects := [...]float64{266.0, 32767.0, 0.0, 1024.0}

	for i := 0; i < len(inputs); i++ {
		f, err = strToFloat(inputs[i], bases[i])
		if err != nil {
			t.Logf("An error was not expected for value %s and base %d: %s", inputs[i], bases[i], err)
			t.Fail()
		}
		if f != expects[i] {
			t.Logf("Expected %f but got %f using base %d", expects[i], f, bases[i])
			t.Fail()
		}
	}
}

func TestStrToFloat_errors(t *testing.T) {
	var err error

	inputs := [...]string{"nope", "FFFFFF"}
	bases := [...]int{16, 16}

	for i := 0; i < len(inputs); i++ {
		_, err = strToFloat(inputs[i], bases[i])
		if err == nil {
			t.Logf("An error was expected for value %s using base %d", inputs[i], bases[i])
			t.Fail()
		}
	}
}

func TestParseValue_ok(t *testing.T) {
	var (
		f   float64
		err error
	)

	inputs := [...]string{
		"OK", "LOW",
		"ON", "OFF",
		"012A", "ABCD",
		"012A", "ABCD",
		"123", "0",
	}
	kinds := [...]string{
		"battery", "battery",
		"onoff", "onoff",
		"hex", "hex",
		"hex_div10", "hex_div10",
		"int", "int",
	}
	expects := [...]float64{
		1.0, 0.0,
		1.0, 0.0,
		298.0, 43981.0,
		29.8, 4398.1,
		123.0, 0.0,
	}

	for i := 0; i < len(inputs); i++ {
		f, err = parseValue(inputs[i], kinds[i])
		if err != nil {
			t.Logf("An error was not expected for value %s and kind %s: %s", inputs[i], kinds[i], err)
			t.Fail()
		}
		if f != expects[i] {
			t.Logf("Expected %f but got %f with kind %s", expects[i], f, kinds[i])
			t.Fail()
		}
	}
}

func TestParseValue_errors(t *testing.T) {
	var err error

	inputs := [...]string{
		"KO",
		"NO",
		"XYZ",
		"XYZ",
		"1A",
		"1234",
	}
	kinds := [...]string{
		"battery",
		"onoff",
		"hex",
		"hex_div10",
		"int",
		"fake",
	}

	for i := 0; i < len(inputs); i++ {
		_, err = parseValue(inputs[i], kinds[i])
		if err == nil {
			t.Logf("An error was expected for value %s and kind %s", inputs[i], kinds[i])
			t.Fail()
		}
	}
}
