package ldap

import (
	"crypto/tls"
	"fmt"
	"strings"

	"modelmatrix-server/pkg/config"
	"modelmatrix-server/pkg/logger"

	"github.com/go-ldap/ldap/v3"
)

// User represents an LDAP user
type User struct {
	DN       string
	UID      string
	Email    string
	FullName string
	Groups   []string
}

// Client defines the LDAP client interface
type Client interface {
	Authenticate(username, password string) (*User, error)
	GetUserGroups(userDN string) ([]string, error)
	GetUser(username string) (*User, error)
	HealthCheck() error
	Close()
}

// LDAPClient implements the Client interface
type LDAPClient struct {
	config *config.LDAPConfig
	conn   *ldap.Conn
}

// NewClient creates a new LDAP client
func NewClient(cfg *config.LDAPConfig) (Client, error) {
	client := &LDAPClient{
		config: cfg,
	}

	if err := client.connect(); err != nil {
		return nil, err
	}

	logger.Info("LDAP client initialized successfully")
	return client, nil
}

// connect establishes connection to LDAP server
func (c *LDAPClient) connect() error {
	var err error
	address := c.config.Address()

	if c.config.UseTLS {
		c.conn, err = ldap.DialTLS("tcp", address, &tls.Config{InsecureSkipVerify: true})
	} else {
		c.conn, err = ldap.Dial("tcp", address)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to LDAP server: %w", err)
	}

	// Bind with service account
	if err := c.conn.Bind(c.config.BindDN, c.config.BindPassword); err != nil {
		return fmt.Errorf("failed to bind to LDAP server: %w", err)
	}

	return nil
}

// reconnect attempts to reconnect to LDAP server
func (c *LDAPClient) reconnect() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return c.connect()
}

// Authenticate authenticates a user against LDAP
func (c *LDAPClient) Authenticate(username, password string) (*User, error) {
	// First, find the user
	user, err := c.GetUser(username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Create a new connection for user bind
	var userConn *ldap.Conn
	if c.config.UseTLS {
		userConn, err = ldap.DialTLS("tcp", c.config.Address(), &tls.Config{InsecureSkipVerify: true})
	} else {
		userConn, err = ldap.Dial("tcp", c.config.Address())
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP: %w", err)
	}
	defer userConn.Close()

	// Bind with user credentials to verify password
	if err := userConn.Bind(user.DN, password); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	// Get user groups
	groups, err := c.GetUserGroups(user.DN)
	if err != nil {
		logger.Warn("Failed to get user groups: %v", err)
		groups = []string{}
	}
	user.Groups = groups

	return user, nil
}

// GetUser retrieves user information from LDAP
func (c *LDAPClient) GetUser(username string) (*User, error) {
	// Build search filter
	filter := fmt.Sprintf(c.config.UserFilter, ldap.EscapeFilter(username))

	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"dn", "uid", "mail", "cn", "displayName"},
		nil,
	)

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		// Try to reconnect and retry
		if err := c.reconnect(); err != nil {
			return nil, fmt.Errorf("LDAP reconnect failed: %w", err)
		}
		result, err = c.conn.Search(searchRequest)
		if err != nil {
			return nil, fmt.Errorf("LDAP search failed: %w", err)
		}
	}

	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("user not found: %s", username)
	}

	entry := result.Entries[0]
	user := &User{
		DN:  entry.DN,
		UID: entry.GetAttributeValue("uid"),
	}

	// Try different attributes for email
	if email := entry.GetAttributeValue("mail"); email != "" {
		user.Email = email
	}

	// Try different attributes for full name
	if cn := entry.GetAttributeValue("cn"); cn != "" {
		user.FullName = cn
	} else if displayName := entry.GetAttributeValue("displayName"); displayName != "" {
		user.FullName = displayName
	}

	return user, nil
}

// GetUserGroups retrieves groups for a user
func (c *LDAPClient) GetUserGroups(userDN string) ([]string, error) {
	// Build group search filter - LLDAP uses uniqueMember, standard LDAP uses member
	filter := fmt.Sprintf("(|(member=%s)(uniqueMember=%s))", ldap.EscapeFilter(userDN), ldap.EscapeFilter(userDN))

	// Search in ou=groups
	groupBaseDN := "ou=groups," + c.config.BaseDN

	logger.Debug("LDAP GetUserGroups: userDN=%s, filter=%s, baseDN=%s", userDN, filter, groupBaseDN)

	searchRequest := ldap.NewSearchRequest(
		groupBaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"cn"},
		nil,
	)

	result, err := c.conn.Search(searchRequest)
	if err != nil {
		logger.Warn("LDAP group search error: %v, attempting reconnect", err)
		// Try reconnect
		if err := c.reconnect(); err != nil {
			return nil, fmt.Errorf("LDAP reconnect failed: %w", err)
		}
		result, err = c.conn.Search(searchRequest)
		if err != nil {
			return nil, fmt.Errorf("LDAP group search failed: %w", err)
		}
	}

	var groups []string
	for _, entry := range result.Entries {
		logger.Debug("LDAP group entry: DN=%s", entry.DN)
		if cn := entry.GetAttributeValue("cn"); cn != "" {
			groups = append(groups, cn)
		}
	}

	logger.Debug("LDAP GetUserGroups result: %v", groups)
	return groups, nil
}

// HealthCheck checks LDAP server connectivity
func (c *LDAPClient) HealthCheck() error {
	// Try a simple search to verify connection
	searchRequest := ldap.NewSearchRequest(
		c.config.BaseDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)

	_, err := c.conn.Search(searchRequest)
	if err != nil {
		// Try to reconnect
		if err := c.reconnect(); err != nil {
			return fmt.Errorf("LDAP health check failed: %w", err)
		}
	}

	return nil
}

// Close closes the LDAP connection
func (c *LDAPClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// HasGroup checks if user has a specific group
func HasGroup(user *User, group string) bool {
	for _, g := range user.Groups {
		if strings.EqualFold(g, group) {
			return true
		}
	}
	return false
}

// HasAnyGroup checks if user has any of the specified groups
func HasAnyGroup(user *User, groups []string) bool {
	for _, g := range groups {
		if HasGroup(user, g) {
			return true
		}
	}
	return false
}

// RBAC Groups
const (
	GroupAdmin  = "modelmatrix_admin"
	GroupEditor = "modelmatrix_editor"
	GroupViewer = "modelmatrix_viewer"
)

// IsAdmin checks if user is an admin
func IsAdmin(user *User) bool {
	return HasGroup(user, GroupAdmin)
}

// IsEditor checks if user is an editor (or admin)
func IsEditor(user *User) bool {
	return HasAnyGroup(user, []string{GroupAdmin, GroupEditor})
}

// IsViewer checks if user is a viewer (or editor/admin)
func IsViewer(user *User) bool {
	return HasAnyGroup(user, []string{GroupAdmin, GroupEditor, GroupViewer})
}
