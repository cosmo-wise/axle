package multidb

type DatabaseDriver interface{ Open(string) error }
