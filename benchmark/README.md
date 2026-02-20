# nlog Benchmark Suite

Diese umfassende Benchmark-Suite testet die Performance verschiedener Aspekte des nlog-Logging-Pakets.

## Übersicht

Die Benchmark-Suite deckt folgende Bereiche ab:

### 1. Logger-Erstellung
- `BenchmarkLoggerCreation`: Misst die Kosten für das Erstellen eines einfachen Loggers
- `BenchmarkLoggerCreationWithFields`: Misst die Kosten mit vordefinierten Feldern
- `BenchmarkWith`: Misst die Kosten für das Erstellen von Child-Loggern mit `With()`

### 2. Logging-Performance
- `BenchmarkInfoNoFields`: Basis-Logging ohne Felder
- `BenchmarkInfo1Field`: Logging mit 1 Feld
- `BenchmarkInfo5Fields`: Logging mit 5 Feldern
- `BenchmarkInfo10Fields`: Logging mit 10 Feldern
- `BenchmarkDisabledLevel`: Prüft Early-Exit-Optimierung für deaktivierte Log-Level

### 3. Feldtypen
- `BenchmarkFieldTypes`: Testet verschiedene Feldtypen (String, Int, Int64, Float64, Bool, Time, Duration, Error, Any)

### 4. Formatter
- `BenchmarkFormatters`: Vergleicht Text vs. JSON Formatter (mit und ohne Caller-Info)
- `BenchmarkWriterFormatter`: Vergleicht `Format()` vs. `FormatTo()` Methoden

### 5. Handler-Modi
- `BenchmarkSyncVsAsync`: Vergleicht synchrone vs. asynchrone Handler
- `BenchmarkWithCaller`: Misst die Auswirkungen von Caller-Informationen
- `BenchmarkOverflowPolicies`: Testet verschiedene Overflow-Policies (DropNewest, Block)

### 6. Log-Level
- `BenchmarkLogLevels`: Vergleicht verschiedene Log-Level (Debug, Info, Warn, Error)

### 7. Concurrency
- `BenchmarkConcurrentLogging`: Testet Performance bei parallelem Logging (1, 2, 4, 8, 16 Goroutinen)

### 8. Handler-Typen
- `BenchmarkFileHandler`: Testet das Schreiben in eine Datei
- `BenchmarkMultiHandler`: Testet mehrere Handler gleichzeitig

### 9. Speicherpools
- `BenchmarkBufferPool`: Testet Buffer-Pooling-Effizienz
- `BenchmarkEntryPool`: Testet Entry-Pooling

### 10. Realistische Szenarien
- `BenchmarkRealisticScenario`: Simuliert eine Web-Anwendung mit Request-Logging
- `BenchmarkFormattedLogging`: Testet formatierte Logging-Methoden (z.B. `Infof`)
- `BenchmarkContextFields`: Testet Context-Felder mit verschiedenen Anzahlen

### 11. Spezialfälle
- `BenchmarkErrorField`: Testet Error-Felder (mit und ohne Error)
- `BenchmarkLargeMessages`: Testet verschiedene Nachrichtengrößen (50B bis 50KB)

## Benchmarks ausführen

### Alle Benchmarks ausführen
```bash
go test -bench=. -benchmem
```

### Bestimmte Benchmarks ausführen
```bash
go test -bench=BenchmarkInfo -benchmem
go test -bench=BenchmarkFormatters -benchmem
```

### Mit kürzerer Laufzeit für schnelles Testen
```bash
go test -bench=. -benchmem -benchtime=100ms
```

### Mit CPU-Profiling
```bash
go test -bench=. -benchmem -cpuprofile=cpu.prof
```

### Mit Memory-Profiling
```bash
go test -bench=. -benchmem -memprofile=mem.prof
```

## Ergebnisse interpretieren

Beispielausgabe:
```
BenchmarkInfoNoFields-12    13173216    90.79 ns/op    0 B/op    0 allocs/op
```

- `13173216`: Anzahl der Iterationen
- `90.79 ns/op`: Nanosekunden pro Operation
- `0 B/op`: Bytes, die pro Operation allokiert wurden
- `0 allocs/op`: Anzahl der Allokationen pro Operation

## Wichtige Metriken

Beim Benchmarking sollten Sie auf folgende Metriken achten:

1. **Nanosekunden pro Operation**: Niedrigere Werte sind besser
2. **Allokationen pro Operation**: Weniger Allokationen bedeuten weniger GC-Druck
3. **Bytes pro Operation**: Weniger Speicherverbrauch ist besser

## Optimierungsziele

Die Benchmarks helfen bei der Identifizierung von:

- Zero-Allocation Logging-Pfaden
- Optimalen Buffer-Größen
- Performance-Unterschieden zwischen Sync/Async
- Formatter-Effizienz
- Caller-Info Overhead
- Overflow-Policy Auswirkungen

## Kontinuierliche Performance-Überwachung

Führen Sie Benchmarks regelmäßig aus, um:

1. Performance-Regressionen zu erkennen
2. Optimierungen zu validieren
3. Verschiedene Implementierungen zu vergleichen
4. Baseline-Metriken für neue Features zu erstellen

## Vergleich mit früheren Versionen

```bash
# Baseline erstellen
go test -bench=. -benchmem > old.txt

# Nach Änderungen
go test -bench=. -benchmem > new.txt

# Vergleichen (benötigt benchstat: go install golang.org/x/perf/cmd/benchstat@latest)
benchstat old.txt new.txt
```

## Best Practices

1. Führen Sie Benchmarks auf einem ruhigen System aus
2. Schließen Sie andere ressourcenintensive Anwendungen
3. Führen Sie mehrere Durchläufe aus, um Konsistenz zu gewährleisten
4. Verwenden Sie `-benchtime` für längere oder kürzere Tests
5. Dokumentieren Sie Baseline-Ergebnisse für Vergleiche

