package main

type MenuItem struct {
	Title string     `yaml:"name"`
	Icon  string     `yaml:"icon"`
	Cmd   []string   `yaml:"cmd"`
	Items []MenuItem `yaml:"items"`
}

type Config struct {
	Menu []MenuItem `yaml:"menu"`
}
