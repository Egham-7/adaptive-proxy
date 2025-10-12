package builder

func (b *Builder) Port(port string) *Builder {
	b.cfg.Server.Port = port
	return b
}

func (b *Builder) AllowedOrigins(origins string) *Builder {
	b.cfg.Server.AllowedOrigins = origins
	return b
}

func (b *Builder) Environment(env string) *Builder {
	b.cfg.Server.Environment = env
	return b
}

func (b *Builder) LogLevel(level string) *Builder {
	b.cfg.Server.LogLevel = level
	return b
}
