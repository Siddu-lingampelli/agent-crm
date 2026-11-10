package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Parse all declared security overrides strictly; a typo must not silently
// disable a protection or leave an operator believing an override was applied.
func loadSecurityEnvironment(config *SecurityConfig) error {
	value := reflect.ValueOf(config).Elem()
	typ := value.Type()
	for i := 0; i < value.NumField(); i++ {
		name := typ.Field(i).Tag.Get("env")
		raw, present := os.LookupEnv(name)
		if !present {
			continue
		}
		raw = strings.TrimSpace(raw)
		field := value.Field(i)
		var err error
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			var duration time.Duration
			duration, err = time.ParseDuration(raw)
			if err == nil {
				field.SetInt(int64(duration))
			}
		} else {
			switch field.Kind() {
			case reflect.Bool:
				var v bool
				v, err = strconv.ParseBool(raw)
				if err == nil {
					field.SetBool(v)
				}
			case reflect.Int, reflect.Int64:
				var v int64
				v, err = strconv.ParseInt(raw, 10, field.Type().Bits())
				if err == nil {
					field.SetInt(v)
				}
			case reflect.Float64:
				var v float64
				v, err = strconv.ParseFloat(raw, 64)
				if err == nil {
					field.SetFloat(v)
				}
			case reflect.String:
				field.SetString(raw)
			case reflect.Slice:
				parts := []string{}
				for _, part := range strings.Split(raw, ",") {
					if part = strings.TrimSpace(part); part != "" {
						parts = append(parts, part)
					}
				}
				field.Set(reflect.ValueOf(parts))
			}
		}
		if err != nil {
			return fmt.Errorf("invalid %s: %w", name, err)
		}
	}
	return nil
}
