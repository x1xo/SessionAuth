package databases

import (
	"time"
)

type UserInfo struct {
	Id        string      `json:"id" bson:"id"`
	Username  string      `json:"username" bson:"username"`
	Email     string      `json:"email" bson:"email"`
	AvatarURL string      `json:"avatar_url" bson:"avatar_url"`
	Github    GithubUser  `json:"github,omitempty" bson:"github"`
	Discord   DiscordUser `json:"discord,omitempty" bson:"discord"`
	Google    GoogleUser  `json:"google,omitempty" bson:"google"`
	CreatedAt time.Time   `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time   `json:"updated_at" bson:"updated_at"`
}

type UserSession struct {
	Id        string        `json:"id,omitempty" bson:"id"`
	Token     string        `json:"token,omitempty" bson:"token"`
	UserId    string        `json:"user_id" bson:"user_id"`
	UserAgent string        `json:"user_agent" bson:"user_agent"`
	Provider  string        `json:"provider" bson:"provider"`
	IssuedAt  time.Time     `json:"issued_at" bson:"issued_at"`
	ExpiresAt time.Time     `json:"expired_at" bson:"expired_at"`
	IPAddress IPAddressInfo `json:"ip_address" bson:"ip_address"`
}

type IPAddressInfo struct {
	IP      string `json:"ip" bson:"ip"`
	City    string `json:"city" bson:"city"`
	Region  string `json:"region" bson:"region"`
	Country string `json:"country" bson:"country"`
}

type GithubUser struct {
	Username  string    `json:"login,omitempty"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type GithubUserEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

type DiscordUser struct {
	Id            string `json:"id,omitempty"`
	Email         string `json:"email,omitempty"`
	Username      string `json:"username,omitempty"`
	Discriminator string `json:"discriminator,omitempty"`
	AvatarURL     string `json:"avatar_url,omitempty"`
	Avatar        string `json:"avatar,omitempty"`
	Verified      bool   `json:"verified,omitempty"`
}

type GoogleUser struct {
	Id        string `json:"id,omitempty"`
	Email     string `json:"email,omitempty"`
	Username  string `json:"name,omitempty"`
	AvatarURL string `json:"picture,omitempty"`
}
