package postgres

import (
	"fmt"
	"net/url"
	"strings"
)

type ConnInfo struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	RawQuery string
	Extra    map[string]string
}

func ParseConnString(connString string) (ConnInfo, error) {
	connString = strings.TrimSpace(connString)
	if connString == "" {
		return ConnInfo{}, fmt.Errorf("connection string is empty")
	}

	if strings.HasPrefix(connString, "postgres://") ||
		strings.HasPrefix(connString, "postgresql://") {
		return parseURIConnString(connString)
	}

	return parseKeywordConnString(connString)
}

func parseURIConnString(connString string) (ConnInfo, error) {
	u, err := url.Parse(connString)
	if err != nil {
		return ConnInfo{}, fmt.Errorf("parse connection URI: %w", err)
	}

	info := ConnInfo{
		Host:     u.Hostname(),
		Port:     u.Port(),
		Database: strings.TrimPrefix(u.Path, "/"),
		RawQuery: u.RawQuery,
		Extra:    map[string]string{},
	}
	if info.Port == "" {
		info.Port = "5432"
	}

	if u.User != nil {
		info.User = u.User.Username()
		pass, ok := u.User.Password()
		if ok {
			info.Password = pass
		}
	}

	for key, values := range u.Query() {
		if len(values) > 0 {
			info.Extra[key] = values[0]
		}
	}

	if info.Database == "" {
		return ConnInfo{}, fmt.Errorf("database name is missing in connection URI")
	}

	return info, nil
}

func parseKeywordConnString(connString string) (ConnInfo, error) {
	info := ConnInfo{Extra: map[string]string{}}

	for _, part := range strings.Fields(connString) {
		key, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		switch key {
		case "host":
			info.Host = value
		case "port":
			info.Port = value
		case "user":
			info.User = value
		case "password":
			info.Password = value
		case "dbname":
			info.Database = value
		default:
			info.Extra[key] = value
		}
	}

	if info.Host == "" {
		return ConnInfo{}, fmt.Errorf("host is missing in connection string")
	}
	if info.Port == "" {
		info.Port = "5432"
	}
	if info.Database == "" {
		return ConnInfo{}, fmt.Errorf("dbname is missing in connection string")
	}

	return info, nil
}

func (info ConnInfo) WithDatabase(database string) string {
	if database == "" {
		return info.String()
	}

	clone := info
	clone.Database = database
	return clone.String()
}

// WithEndpoint returns a connection string with host, port and database replaced.
func (info ConnInfo) WithEndpoint(host string, port int, database string) string {
	clone := info
	if host = strings.TrimSpace(host); host != "" {
		clone.Host = host
	}
	if port >= 1 && port <= 65535 {
		clone.Port = fmt.Sprintf("%d", port)
	}
	if database = strings.TrimSpace(database); database != "" {
		clone.Database = database
	}
	return clone.String()
}

func (info ConnInfo) String() string {
	return info.keywordString()
}

func (info ConnInfo) keywordString() string {
	parts := []string{
		fmt.Sprintf("host=%s", info.Host),
		fmt.Sprintf("port=%s", info.Port),
		fmt.Sprintf("dbname=%s", info.Database),
	}
	if info.User != "" {
		parts = append(parts, fmt.Sprintf("user=%s", info.User))
	}
	if info.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", info.Password))
	}
	for key, value := range info.Extra {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, " ")
}
