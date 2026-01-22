package client

import (
	"fmt"
	"strings"

	myshoespb "github.com/whywaita/myshoes/api/proto.go"
)

// ConvertResourceType converts myshoes ResourceType enum to shoes-vz resource type string
func ConvertResourceType(rt myshoespb.ResourceType) (string, error) {
	switch rt {
	case myshoespb.ResourceType_Nano:
		return "nano", nil
	case myshoespb.ResourceType_Micro:
		return "micro", nil
	case myshoespb.ResourceType_Small:
		return "small", nil
	case myshoespb.ResourceType_Medium:
		return "medium", nil
	case myshoespb.ResourceType_Large:
		return "large", nil
	case myshoespb.ResourceType_XLarge:
		return "xlarge", nil
	case myshoespb.ResourceType_XLarge2:
		return "xlarge2", nil
	case myshoespb.ResourceType_XLarge3:
		return "xlarge3", nil
	case myshoespb.ResourceType_XLarge4:
		return "xlarge4", nil
	case myshoespb.ResourceType_Unknown:
		return "", fmt.Errorf("unknown resource type")
	default:
		return "", fmt.Errorf("unsupported resource type: %v", rt)
	}
}

// ConvertResourceTypeFromString converts shoes-vz resource type string to myshoes ResourceType enum
func ConvertResourceTypeFromString(s string) (myshoespb.ResourceType, error) {
	switch strings.ToLower(s) {
	case "nano":
		return myshoespb.ResourceType_Nano, nil
	case "micro":
		return myshoespb.ResourceType_Micro, nil
	case "small":
		return myshoespb.ResourceType_Small, nil
	case "medium":
		return myshoespb.ResourceType_Medium, nil
	case "large":
		return myshoespb.ResourceType_Large, nil
	case "xlarge":
		return myshoespb.ResourceType_XLarge, nil
	case "xlarge2":
		return myshoespb.ResourceType_XLarge2, nil
	case "xlarge3":
		return myshoespb.ResourceType_XLarge3, nil
	case "xlarge4":
		return myshoespb.ResourceType_XLarge4, nil
	default:
		return myshoespb.ResourceType_Unknown, fmt.Errorf("unsupported resource type string: %s", s)
	}
}
