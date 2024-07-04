package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mackerelio/mackerel-client-go"
)

func Test_generateCustomMIB(t *testing.T) {

	tests := []struct {
		source   *CustomMIB
		expected *customMIBConfig
		wantErr  bool
	}{
		{
			source: &CustomMIB{
				Mibs: []*MIBwithDisplayName{
					{
						MetricName: "foo.bar",
						MIB:        "1.2.3.4",
					},
				},
			},
			expected: &customMIBConfig{
				customMIBs: []string{"1.2.3.4"},
				metricNameMappedMIBs: map[string]string{
					"custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar": "1.2.3.4",
				},
				graphDefs: &mackerel.GraphDefsParam{
					Name: "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e",
					Metrics: []*mackerel.GraphDefsMetric{
						{
							DisplayName: "foo.bar",
							Name:        "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar",
						},
					},
				},
			},
		},
		{
			source: &CustomMIB{
				Mibs: []*MIBwithDisplayName{
					{
						DisplayName: "foobarbaz",
						MetricName:  "foo.bar",
						MIB:         "1.2.3.4",
					},
				},
			},
			expected: &customMIBConfig{
				customMIBs: []string{"1.2.3.4"},
				metricNameMappedMIBs: map[string]string{
					"custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar": "1.2.3.4",
				},
				graphDefs: &mackerel.GraphDefsParam{
					Name: "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e",
					Metrics: []*mackerel.GraphDefsMetric{
						{
							DisplayName: "foobarbaz",
							Name:        "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar",
						},
					},
				},
			},
		},
		{
			source: &CustomMIB{
				Mibs: []*MIBwithDisplayName{
					{
						MetricName: "foo.bar",
						MIB:        "1.2.3.4",
					},
					{
						MetricName: "foo.baz",
						MIB:        "5.6.7.8",
					},
				},
			},
			expected: &customMIBConfig{
				customMIBs: []string{"1.2.3.4", "5.6.7.8"},
				metricNameMappedMIBs: map[string]string{
					"custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar": "1.2.3.4",
					"custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.baz": "5.6.7.8",
				},
				graphDefs: &mackerel.GraphDefsParam{
					Name: "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e",
					Metrics: []*mackerel.GraphDefsMetric{
						{
							Name:        "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.bar",
							DisplayName: "foo.bar",
						},
						{
							Name:        "custom.custommibs.d41d8cd98f00b204e9800998ecf8427e.foo.baz",
							DisplayName: "foo.baz",
						},
					},
				},
			},
		},
		{
			source: &CustomMIB{
				Mibs: []*MIBwithDisplayName{
					{
						MetricName: "foo.bar",
						MIB:        "1.2.3.4...",
					},
				},
			},
			wantErr: true,
		},
		{
			source: &CustomMIB{
				Mibs: []*MIBwithDisplayName{
					{
						MetricName: "foo.あ.bar",
						MIB:        "1.2.3.4",
					},
				},
			},
			wantErr: true,
		},
	}

	opt := cmp.AllowUnexported(customMIBConfig{})
	for _, tc := range tests {
		actual, err := generateCustomMIB(tc.source)
		if (err != nil) != tc.wantErr {
			t.Error(err)
		}

		if diff := cmp.Diff(actual, tc.expected, opt); diff != "" {
			t.Errorf("value is mismatch (-actual +expected):%s", diff)
		}
	}
}

func Test_convert(t *testing.T) {
	reg := "^(eth|wlan)"

	tests := []struct {
		source   YAMLConfig
		expected *Config
		wantErr  bool
	}{
		{
			source:  YAMLConfig{},
			wantErr: true,
		},
		{
			source: YAMLConfig{
				Community: "public",
			},
			wantErr: true,
		},
		{
			source: YAMLConfig{
				Target: "192.0.2.1",
			},
			wantErr: true,
		},
		{
			source: YAMLConfig{
				Community: "public",
				Target:    "192.0.2.1",
			},
			expected: &Config{
				Community:                     "public",
				Target:                        "192.0.2.1",
				MIBs:                          []string{"ifHCInOctets", "ifHCOutOctets", "ifInDiscards", "ifOutDiscards", "ifInErrors", "ifOutErrors"},
				CustomMIBmetricNameMappedMIBs: map[string]string{},
			},
		},
		{
			source: YAMLConfig{
				Community: "public",
				Target:    "192.0.2.1",
				Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
			},
			expected: &Config{
				Community:                     "public",
				Target:                        "192.0.2.1",
				MIBs:                          []string{"ifHCInOctets", "ifHCOutOctets"},
				CustomMIBmetricNameMappedMIBs: map[string]string{},
			},
		},
		{
			source: YAMLConfig{
				Community: "public",
				Target:    "192.0.2.1",
				Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
				Interface: &Interface{
					Include: &reg,
				},
			},
			expected: &Config{
				Community:                     "public",
				Target:                        "192.0.2.1",
				MIBs:                          []string{"ifHCInOctets", "ifHCOutOctets"},
				CustomMIBmetricNameMappedMIBs: map[string]string{},
				IncludeRegexp:                 regexp.MustCompile(reg),
			},
		},
		{
			source: YAMLConfig{
				Community: "public",
				Target:    "192.0.2.1",
				Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
				Interface: &Interface{
					Exclude: &reg,
				},
			},
			expected: &Config{
				Community:                     "public",
				Target:                        "192.0.2.1",
				MIBs:                          []string{"ifHCInOctets", "ifHCOutOctets"},
				CustomMIBmetricNameMappedMIBs: map[string]string{},
				ExcludeRegexp:                 regexp.MustCompile(reg),
			},
		},
		{
			source: YAMLConfig{
				Community: "public",
				Target:    "192.0.2.1",
				Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
				Interface: &Interface{
					Include: &reg,
					Exclude: &reg,
				},
			},
			wantErr: true,
		},
		{
			source: YAMLConfig{
				Community: "public",
				Target:    "192.0.2.1",
				Mibs:      []string{"^o^"},
			},
			wantErr: true,
		},
		{
			source: YAMLConfig{
				Community: "public",
				Target:    "192.0.2.1",
				Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
				Mackerel: &Mackerel{
					ApiKey:            "cat",
					HostID:            "panda",
					Name:              "dog",
					IgnoreNetworkInfo: true,
				},
			},
			expected: &Config{
				Community:                     "public",
				Target:                        "192.0.2.1",
				MIBs:                          []string{"ifHCInOctets", "ifHCOutOctets"},
				CustomMIBmetricNameMappedMIBs: map[string]string{},
				Mackerel: &Mackerel{
					HostID:            "panda",
					ApiKey:            "cat",
					Name:              "dog",
					IgnoreNetworkInfo: true,
				},
			},
		},
		{
			source: YAMLConfig{
				Community: "public",
				Target:    "192.0.2.1",
				Mibs:      []string{"ifHCInOctets", "ifHCOutOctets"},
				CustomMibs: []*CustomMIB{
					{
						DisplayName: "zoo",
						Unit:        "float",
						Mibs: []*MIBwithDisplayName{
							{
								DisplayName: "foobar",
								MetricName:  "foo.bar",
								MIB:         "1.2.34.56",
							},
						},
					},
				},
			},
			expected: &Config{
				Community: "public",
				Target:    "192.0.2.1",
				MIBs:      []string{"ifHCInOctets", "ifHCOutOctets"},
				CustomMIBmetricNameMappedMIBs: map[string]string{
					"custom.custommibs.d2cbe65f53da8607e64173c1a83394fe.foo.bar": "1.2.34.56",
				},
				CustomMIBs: []string{"1.2.34.56"},
				CustomMIBsGraphDefs: []*mackerel.GraphDefsParam{
					{
						Name:        "custom.custommibs.d2cbe65f53da8607e64173c1a83394fe",
						DisplayName: "zoo",
						Unit:        "float",
						Metrics: []*mackerel.GraphDefsMetric{
							{
								Name:        "custom.custommibs.d2cbe65f53da8607e64173c1a83394fe.foo.bar",
								DisplayName: "foobar",
							},
						},
					},
				},
			},
		},
	}

	opt1 := cmpopts.SortSlices(func(i, j string) bool { return i < j })

	opt2 := cmp.Comparer(func(x, y *regexp.Regexp) bool {
		if x == nil || y == nil {
			return x == y
		}

		return fmt.Sprint(x) == fmt.Sprint(y)
	})

	for _, tc := range tests {
		actual, err := convert(tc.source)
		if (err != nil) != tc.wantErr {
			t.Error(err)
		}

		if diff := cmp.Diff(actual, tc.expected, opt1, opt2); diff != "" {
			t.Errorf("value is mismatch (-actual +expected):%s", diff)
		}
	}
}

func Test_Config_Save(t *testing.T) {
	dir := t.TempDir()

	loadedFilename = filepath.Join(dir, "data.yml")

	var perm fs.FileMode = 0644
	err := os.WriteFile(loadedFilename, []byte("---\nmackerel:\n  name: 1234"), perm)
	if err != nil {
		t.Error(err)
	}

	c := &Config{
		Mackerel: &Mackerel{},
	}

	err = c.Save("123456")
	if err != nil {
		t.Error(err)
	}

	actual, err := os.ReadFile(loadedFilename)
	if err != nil {
		t.Error(err)
	}

	expected := []byte(`community: ""
target: ""
mackerel:
    host-id: "123456"
    x-api-key: ""
    name: "1234"
`)

	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("value is mismatch (-actual +expected):%s", diff)
	}

}
