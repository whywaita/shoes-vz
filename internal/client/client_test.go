package client

import (
	"os"
	"testing"

	myshoespb "github.com/whywaita/myshoes/api/proto.go"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		wantErr  bool
		wantAddr string
	}{
		{
			name:     "valid config",
			envValue: "localhost:50051",
			wantErr:  false,
			wantAddr: "localhost:50051",
		},
		{
			name:     "empty config",
			envValue: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv(EnvShoesVzServerAddr, tt.envValue)
			} else {
				os.Unsetenv(EnvShoesVzServerAddr)
			}
			defer os.Unsetenv(EnvShoesVzServerAddr)

			// Execute
			config, err := LoadConfig()

			// Assert
			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadConfig() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if config.ServerAddr != tt.wantAddr {
				t.Errorf("LoadConfig() ServerAddr = %v, want %v", config.ServerAddr, tt.wantAddr)
			}
		})
	}
}

func TestConvertResourceType(t *testing.T) {
	tests := []struct {
		name    string
		rt      myshoespb.ResourceType
		want    string
		wantErr bool
	}{
		{
			name:    "nano",
			rt:      myshoespb.ResourceType_Nano,
			want:    "nano",
			wantErr: false,
		},
		{
			name:    "micro",
			rt:      myshoespb.ResourceType_Micro,
			want:    "micro",
			wantErr: false,
		},
		{
			name:    "small",
			rt:      myshoespb.ResourceType_Small,
			want:    "small",
			wantErr: false,
		},
		{
			name:    "medium",
			rt:      myshoespb.ResourceType_Medium,
			want:    "medium",
			wantErr: false,
		},
		{
			name:    "large",
			rt:      myshoespb.ResourceType_Large,
			want:    "large",
			wantErr: false,
		},
		{
			name:    "xlarge",
			rt:      myshoespb.ResourceType_XLarge,
			want:    "xlarge",
			wantErr: false,
		},
		{
			name:    "xlarge2",
			rt:      myshoespb.ResourceType_XLarge2,
			want:    "xlarge2",
			wantErr: false,
		},
		{
			name:    "xlarge3",
			rt:      myshoespb.ResourceType_XLarge3,
			want:    "xlarge3",
			wantErr: false,
		},
		{
			name:    "xlarge4",
			rt:      myshoespb.ResourceType_XLarge4,
			want:    "xlarge4",
			wantErr: false,
		},
		{
			name:    "unknown",
			rt:      myshoespb.ResourceType_Unknown,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertResourceType(tt.rt)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ConvertResourceType() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ConvertResourceType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ConvertResourceType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertResourceTypeFromString(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    myshoespb.ResourceType
		wantErr bool
	}{
		{
			name:    "nano",
			s:       "nano",
			want:    myshoespb.ResourceType_Nano,
			wantErr: false,
		},
		{
			name:    "NANO uppercase",
			s:       "NANO",
			want:    myshoespb.ResourceType_Nano,
			wantErr: false,
		},
		{
			name:    "micro",
			s:       "micro",
			want:    myshoespb.ResourceType_Micro,
			wantErr: false,
		},
		{
			name:    "small",
			s:       "small",
			want:    myshoespb.ResourceType_Small,
			wantErr: false,
		},
		{
			name:    "medium",
			s:       "medium",
			want:    myshoespb.ResourceType_Medium,
			wantErr: false,
		},
		{
			name:    "large",
			s:       "large",
			want:    myshoespb.ResourceType_Large,
			wantErr: false,
		},
		{
			name:    "xlarge",
			s:       "xlarge",
			want:    myshoespb.ResourceType_XLarge,
			wantErr: false,
		},
		{
			name:    "xlarge2",
			s:       "xlarge2",
			want:    myshoespb.ResourceType_XLarge2,
			wantErr: false,
		},
		{
			name:    "xlarge3",
			s:       "xlarge3",
			want:    myshoespb.ResourceType_XLarge3,
			wantErr: false,
		},
		{
			name:    "xlarge4",
			s:       "xlarge4",
			want:    myshoespb.ResourceType_XLarge4,
			wantErr: false,
		},
		{
			name:    "invalid",
			s:       "invalid",
			want:    myshoespb.ResourceType_Unknown,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertResourceTypeFromString(tt.s)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ConvertResourceTypeFromString() error = nil, wantErr %v", tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ConvertResourceTypeFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ConvertResourceTypeFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
