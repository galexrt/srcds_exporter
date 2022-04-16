/*
Copyright 2022 Alexander Trost <galexrt@googlemail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"time"
)

// Config Config file structure
type Config struct {
	Options Options           `yaml:"options"`
	Servers map[string]Server `yaml:"servers"`
}

// Options Options structure
type Options struct {
	ConnectTimeout       time.Duration `yaml:"connectTimeout"`
	CacheExpiration      time.Duration `yaml:"cacheExpiration"`
	CacheCleanupInterval time.Duration `yaml:"cacheCleanupInterval"`
}

// Server Server structure
type Server struct {
	Address      string    `yaml:"address"`
	RCONPassword string    `yaml:"rconPassword"`
	Mode         QueryMode `yaml:"mode"`
}

// QueryMode which mode to talk to a server with
type QueryMode string

const (
	RCONMode        QueryMode = "RCON"
	ServerQueryMode QueryMode = "ServerQuery"
)
