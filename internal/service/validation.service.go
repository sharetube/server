package service

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var UsernameRule = []validation.Rule{
	validation.Required,
	validation.Length(0, 20),
}

var ColorRule = []validation.Rule{
	validation.Required,
	is.HexColor,
}

var AvatarUrlRule = []validation.Rule{
	is.URL,
}

var VideoUrlRule = []validation.Rule{
	validation.Required,
	validation.Match(regexp.MustCompile("^[a-zA-Z0-9_-]{11}$")),
}

var VideoIdRule = []validation.Rule{
	validation.Required,
}

var RoomIdRule = []validation.Rule{
	validation.Required,
	validation.Match(regexp.MustCompile("^[a-zA-Z0-9.-]{8}$")),
}

var MemberIdRule = []validation.Rule{
	validation.Required,
	is.UUIDv4,
}
