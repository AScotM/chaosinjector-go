package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

type ChaosStrategy string

const (
	Latency          ChaosStrategy = "latency"
	Failure          ChaosStrategy = "failure"
	GatewayOutage    ChaosStrategy = "gateway_outage"
	MemoryLeak       ChaosStrategy = "memory_leak"
	CPUSpike         ChaosStrategy = "cpu_spike"
	NetworkPartition ChaosStrategy = "network_partition"
	Random           ChaosStrategy = "random"
)

type ChaosIntensity string

const (
	Low     ChaosIntensity = "low"
	Medium  ChaosIntensity = "medium"
	High    ChaosIntensity = "high"
	Extreme ChaosIntensity = "extreme"
)

type ChaosConfig struct {
	LatencyProbability          float64
	FailureProbability          float64
	GatewayOutageProbability    float64
	MemoryLeakProbability       float64
	CPUSpikeProbability         float64
	NetworkPartitionProbability float64
	MaxLatency                  float64
	MaxMemoryLeakMB             int
	MaxCPUSpikeDuration         float64
	EnabledStrategies           []ChaosStrategy
	Intensity                   ChaosIntensity
	GatewayOutageDuration       float64
	NetworkPartitionDuration    float64
}

type ChaosEvent struct {
	EventID           string
	Strategy          ChaosStrategy
	Intensity         ChaosIntensity
	Timestamp         time.Time
	Description       string
	AffectedComponent string
	Duration          float64
	Impact            string
}

type StrategyStats struct {
	Count          int64
	TotalDuration  float64
	LastOccurrence time.Time
}

type ChaosMonitor struct {
	events        []ChaosEvent
	eventCount    int64
	startTime     time.Time
	strategyStats map[ChaosStrategy]*StrategyStats
	mu            sync.RWMutex
}

type MemoryLeakSimulator struct {
	maxLeakMB   int
	leakedData  [][]byte
	totalLeaked int
	mu          sync.Mutex
}

type CPUSpikeSimulator struct {
	activeSpikes        int
	maxConcurrentSpikes int
	mu                  sync.Mutex
}

type ChaosInjector struct {
	config              *ChaosConfig
	monitor             *ChaosMonitor
	memoryLeakSimulator *MemoryLeakSimulator
	cpuSpikeSimulator   *CPUSpikeSimulator
	gatewayOutages      map[string]time.Time
	networkPartitions   map[string]time.Time
	activeChaos         bool
	injectionSemaphore  chan struct{}
	mu                  sync.RWMutex
}

func NewChaosConfig() *ChaosConfig {
	config := &ChaosConfig{
		LatencyProbability:          0.03,
		FailureProbability:          0.02,
		GatewayOutageProbability:    0.02,
		MemoryLeakProbability:       0.01,
		CPUSpikeProbability:         0.01,
		NetworkPartitionProbability: 0.005,
		MaxLatency:                  2.0,
		MaxMemoryLeakMB:             100,
		MaxCPUSpikeDuration:         5.0,
		EnabledStrategies:           []ChaosStrategy{Latency, Failure, GatewayOutage},
		Intensity:                   Medium,
		GatewayOutageDuration:       30.0,
		NetworkPartitionDuration:    60.0,
	}
	config.ApplyIntensityMultiplier()
	return config
}

func (cc *ChaosConfig) ApplyIntensityMultiplier() {
	multiplier := map[ChaosIntensity]float64{
		Low:     0.5,
		Medium:  1.0,
		High:    2.0,
		Extreme: 4.0,
	}[cc.Intensity]

	cc.LatencyProbability = min(cc.LatencyProbability*multiplier, 1.0)
	cc.FailureProbability = min(cc.FailureProbability*multiplier, 1.0)
	cc.GatewayOutageProbability = min(cc.GatewayOutageProbability*multiplier, 1.0)
	cc.MemoryLeakProbability = min(cc.MemoryLeakProbability*multiplier, 1.0)
	cc.CPUSpikeProbability = min(cc.CPUSpikeProbability*multiplier, 1.0)
	cc.NetworkPartitionProbability = min(cc.NetworkPartitionProbability*multiplier, 1.0)
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func NewChaosMonitor() *ChaosMonitor {
	return &ChaosMonitor{
		events:        make([]ChaosEvent, 0),
		startTime:     time.Now(),
		strategyStats: make(map[ChaosStrategy]*StrategyStats),
	}
}

func (cm *ChaosMonitor) RecordEvent(event ChaosEvent) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.events = append(cm.events, event)
	cm.eventCount++

	if cm.strategyStats[event.Strategy] == nil {
		cm.strategyStats[event.Strategy] = &StrategyStats{}
	}

	stats := cm.strategyStats[event.Strategy]
	stats.Count++
	stats.TotalDuration += event.Duration
	stats.LastOccurrence = event.Timestamp

	log.Printf("Chaos event recorded: %s", event.Description)
}

func (cm *ChaosMonitor) GetMetrics() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	runtime := time.Since(cm.startTime).Seconds()
	eventsByStrategy := make(map[string]int)
	strategyDetails := make(map[string]interface{})

	for strategy, stats := range cm.strategyStats {
		eventsByStrategy[string(strategy)] = int(stats.Count)
		avgDuration := 0.0
		if stats.Count > 0 {
			avgDuration = stats.TotalDuration / float64(stats.Count)
		}
		strategyDetails[string(strategy)] = map[string]interface{}{
			"total_events":     stats.Count,
			"average_duration": avgDuration,
			"last_occurrence":  stats.LastOccurrence.Format(time.RFC3339),
		}
	}

	recentEvents := make([]ChaosEvent, 0)
	if len(cm.events) > 10 {
		recentEvents = cm.events[len(cm.events)-10:]
	} else {
		recentEvents = cm.events
	}

	eventsPerMinute := 0.0
	if runtime > 0 {
		eventsPerMinute = float64(cm.eventCount) / (runtime / 60)
	}

	return map[string]interface{}{
		"total_events":       cm.eventCount,
		"runtime_seconds":    runtime,
		"events_per_minute":  eventsPerMinute,
		"events_by_strategy": eventsByStrategy,
		"strategy_details":   strategyDetails,
		"recent_events":      recentEvents,
	}
}

func (cm *ChaosMonitor) ClearEvents() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.events = make([]ChaosEvent, 0)
	cm.eventCount = 0
	cm.strategyStats = make(map[ChaosStrategy]*StrategyStats)
	cm.startTime = time.Now()
}

func NewMemoryLeakSimulator(maxLeakMB int) *MemoryLeakSimulator {
	return &MemoryLeakSimulator{
		maxLeakMB:  maxLeakMB,
		leakedData: make([][]byte, 0),
	}
}

func (mls *MemoryLeakSimulator) InjectMemoryLeak(sizeMB int) *ChaosEvent {
	mls.mu.Lock()
	defer mls.mu.Unlock()

	leakSize := sizeMB
	if mls.totalLeaked+leakSize > mls.maxLeakMB {
		leakSize = mls.maxLeakMB - mls.totalLeaked
	}

	if leakSize <= 0 {
		return &ChaosEvent{
			EventID:           fmt.Sprintf("memleak_%d", time.Now().Unix()),
			Strategy:          MemoryLeak,
			Intensity:         Low,
			Timestamp:         time.Now(),
			Description:       "Memory leak skipped - maximum reached",
			AffectedComponent: "memory",
			Impact:            "No additional memory allocated",
		}
	}

	data := make([]byte, leakSize*1024*1024)
	for i := range data {
		data[i] = 'X'
	}

	mls.leakedData = append(mls.leakedData, data)
	mls.totalLeaked += leakSize

	return &ChaosEvent{
		EventID:           fmt.Sprintf("memleak_%d", time.Now().Unix()),
		Strategy:          MemoryLeak,
		Intensity:         Medium,
		Timestamp:         time.Now(),
		Description:       fmt.Sprintf("Injected memory leak of %dMB", leakSize),
		AffectedComponent: "memory",
		Impact:            fmt.Sprintf("Allocated %dMB of memory (total: %dMB)", leakSize, mls.totalLeaked),
	}
}

func (mls *MemoryLeakSimulator) GetMemoryUsage() map[string]interface{} {
	mls.mu.Lock()
	defer mls.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"total_leaked_mb": mls.totalLeaked,
		"leak_chunks":     len(mls.leakedData),
		"process_rss_mb":  float64(m.Sys) / 1024 / 1024,
		"process_vms_mb":  float64(m.HeapSys) / 1024 / 1024,
	}
}

func (mls *MemoryLeakSimulator) Cleanup() {
	mls.mu.Lock()
	defer mls.mu.Unlock()

	mls.leakedData = make([][]byte, 0)
	mls.totalLeaked = 0
	runtime.GC()
}

func NewCPUSpikeSimulator() *CPUSpikeSimulator {
	return &CPUSpikeSimulator{
		maxConcurrentSpikes: 3,
	}
}

func (css *CPUSpikeSimulator) InjectCPUSpike(duration float64) *ChaosEvent {
	css.mu.Lock()
	if css.activeSpikes >= css.maxConcurrentSpikes {
		css.mu.Unlock()
		return &ChaosEvent{
			EventID:           fmt.Sprintf("cpu_skip_%d", time.Now().Unix()),
			Strategy:          CPUSpike,
			Intensity:         Low,
			Timestamp:         time.Now(),
			Description:       "CPU spike skipped - too many active spikes",
			AffectedComponent: "cpu",
			Impact:            "No CPU spike injected due to resource limits",
		}
	}
	css.activeSpikes++
	css.mu.Unlock()

	defer func() {
		css.mu.Lock()
		css.activeSpikes--
		css.mu.Unlock()
	}()

	startTime := time.Now()
	iterations := 0

	for time.Since(startTime).Seconds() < duration {
		result := 0
		for i := 0; i < 1000000; i++ {
			result += i * i
		}
		iterations++
		_ = result
		time.Sleep(1 * time.Millisecond)
	}

	return &ChaosEvent{
		EventID:           fmt.Sprintf("cpu_%d", time.Now().Unix()),
		Strategy:          CPUSpike,
		Intensity:         Medium,
		Timestamp:         time.Now(),
		Description:       fmt.Sprintf("Injected CPU spike for %.2fs", duration),
		AffectedComponent: "cpu",
		Duration:          duration,
		Impact:            fmt.Sprintf("CPU intensive computation for %.2fs (%d iterations)", duration, iterations),
	}
}

func (css *CPUSpikeSimulator) GetActiveSpikes() int {
	css.mu.Lock()
	defer css.mu.Unlock()
	return css.activeSpikes
}

func NewChaosInjector(config *ChaosConfig) *ChaosInjector {
	if config == nil {
		config = NewChaosConfig()
	}

	return &ChaosInjector{
		config:              config,
		monitor:             NewChaosMonitor(),
		memoryLeakSimulator: NewMemoryLeakSimulator(config.MaxMemoryLeakMB),
		cpuSpikeSimulator:   NewCPUSpikeSimulator(),
		gatewayOutages:      make(map[string]time.Time),
		networkPartitions:   make(map[string]time.Time),
		injectionSemaphore:  make(chan struct{}, 5),
	}
}

func (ci *ChaosInjector) InjectPaymentChaos(paymentID string) []ChaosEvent {
	if !ci.isChaosActive() {
		return nil
	}

	select {
	case ci.injectionSemaphore <- struct{}{}:
		defer func() { <-ci.injectionSemaphore }()
	default:
		log.Printf("Too many concurrent chaos injections for payment %s", paymentID)
		return nil
	}

	var events []ChaosEvent
	for _, strategy := range ci.config.EnabledStrategies {
		event := ci.applyStrategy(strategy, paymentID)
		if event != nil {
			events = append(events, *event)
			ci.monitor.RecordEvent(*event)
		}
	}
	return events
}

func (ci *ChaosInjector) applyStrategy(strategy ChaosStrategy, paymentID string) *ChaosEvent {
	switch strategy {
	case Latency:
		if rand.Float64() < ci.config.LatencyProbability {
			return ci.injectLatency(paymentID)
		}
	case Failure:
		if rand.Float64() < ci.config.FailureProbability {
			return ci.injectFailure(paymentID)
		}
	case GatewayOutage:
		if rand.Float64() < ci.config.GatewayOutageProbability {
			return ci.injectGatewayOutage(paymentID)
		}
	case MemoryLeak:
		if rand.Float64() < ci.config.MemoryLeakProbability {
			return ci.injectMemoryLeak(paymentID)
		}
	case CPUSpike:
		if rand.Float64() < ci.config.CPUSpikeProbability {
			return ci.injectCPUSpike(paymentID)
		}
	case NetworkPartition:
		if rand.Float64() < ci.config.NetworkPartitionProbability {
			return ci.injectNetworkPartition(paymentID)
		}
	case Random:
		availableStrategies := []ChaosStrategy{Latency, Failure, GatewayOutage, MemoryLeak, CPUSpike, NetworkPartition}
		randomStrategy := availableStrategies[rand.Intn(len(availableStrategies))]
		return ci.applyStrategy(randomStrategy, paymentID)
	}
	return nil
}

func (ci *ChaosInjector) injectLatency(paymentID string) *ChaosEvent {
	latency := rand.Float64() * ci.config.MaxLatency
	time.Sleep(time.Duration(latency * float64(time.Second)))

	return &ChaosEvent{
		EventID:           fmt.Sprintf("latency_%d", time.Now().Unix()),
		Strategy:          Latency,
		Intensity:         ci.config.Intensity,
		Timestamp:         time.Now(),
		Description:       fmt.Sprintf("Injected %.3fs latency", latency),
		AffectedComponent: "network",
		Duration:          latency,
		Impact:            fmt.Sprintf("Payment %s delayed by %.3fs", paymentID, latency),
	}
}

func (ci *ChaosInjector) injectFailure(paymentID string) *ChaosEvent {
	failureTypes := []string{"timeout", "connection_reset", "protocol_error", "server_error"}
	failureType := failureTypes[rand.Intn(len(failureTypes))]

	return &ChaosEvent{
		EventID:           fmt.Sprintf("failure_%d", time.Now().Unix()),
		Strategy:          Failure,
		Intensity:         ci.config.Intensity,
		Timestamp:         time.Now(),
		Description:       fmt.Sprintf("Injected %s failure", failureType),
		AffectedComponent: "payment_processing",
		Impact:            fmt.Sprintf("Payment %s affected by %s", paymentID, failureType),
	}
}

func (ci *ChaosInjector) injectGatewayOutage(paymentID string) *ChaosEvent {
	gateways := []string{"Stripe", "PayPal", "Square", "Adyen"}
	affectedGateway := gateways[rand.Intn(len(gateways))]
	outageEnd := time.Now().Add(time.Duration(ci.config.GatewayOutageDuration * float64(time.Second)))

	ci.mu.Lock()
	ci.gatewayOutages[affectedGateway] = outageEnd
	ci.mu.Unlock()

	return &ChaosEvent{
		EventID:           fmt.Sprintf("outage_%d", time.Now().Unix()),
		Strategy:          GatewayOutage,
		Intensity:         ci.config.Intensity,
		Timestamp:         time.Now(),
		Description:       fmt.Sprintf("Simulated outage for %s gateway", affectedGateway),
		AffectedComponent: fmt.Sprintf("gateway_%s", affectedGateway),
		Duration:          ci.config.GatewayOutageDuration,
		Impact:            fmt.Sprintf("Gateway %s marked as unavailable", affectedGateway),
	}
}

func (ci *ChaosInjector) injectMemoryLeak(paymentID string) *ChaosEvent {
	leakSize := rand.Intn(10) + 1
	return ci.memoryLeakSimulator.InjectMemoryLeak(leakSize)
}

func (ci *ChaosInjector) injectCPUSpike(paymentID string) *ChaosEvent {
	duration := rand.Float64() * ci.config.MaxCPUSpikeDuration
	return ci.cpuSpikeSimulator.InjectCPUSpike(duration)
}

func (ci *ChaosInjector) injectNetworkPartition(paymentID string) *ChaosEvent {
	components := []string{"database", "cache", "external_api", "authentication_service"}
	affectedComponent := components[rand.Intn(len(components))]
	partitionEnd := time.Now().Add(time.Duration(ci.config.NetworkPartitionDuration * float64(time.Second)))

	ci.mu.Lock()
	ci.networkPartitions[affectedComponent] = partitionEnd
	ci.mu.Unlock()

	return &ChaosEvent{
		EventID:           fmt.Sprintf("partition_%d", time.Now().Unix()),
		Strategy:          NetworkPartition,
		Intensity:         ci.config.Intensity,
		Timestamp:         time.Now(),
		Description:       fmt.Sprintf("Simulated network partition for %s", affectedComponent),
		AffectedComponent: affectedComponent,
		Duration:          ci.config.NetworkPartitionDuration,
		Impact:            fmt.Sprintf("Component %s isolated from network", affectedComponent),
	}
}

func (ci *ChaosInjector) isChaosActive() bool {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	return ci.activeChaos
}

func (ci *ChaosInjector) GetSuccessRateModifier() float64 {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	modifier := 1.0

	activeOutages := 0
	for _, endTime := range ci.gatewayOutages {
		if endTime.After(time.Now()) {
			activeOutages++
		}
	}
	modifier -= 0.1 * float64(activeOutages)

	activePartitions := 0
	for _, endTime := range ci.networkPartitions {
		if endTime.After(time.Now()) {
			activePartitions++
		}
	}
	modifier -= 0.05 * float64(activePartitions)

	if modifier < 0.3 {
		return 0.3
	}
	return modifier
}

func (ci *ChaosInjector) IsGatewayAvailable(gatewayName string) bool {
	ci.mu.RLock()
	defer ci.mu.RUnlock()

	outageEnd, exists := ci.gatewayOutages[gatewayName]
	if !exists {
		return true
	}
	return !outageEnd.After(time.Now())
}

func (ci *ChaosInjector) CleanupExpiredChaos() {
	ci.mu.Lock()
	defer ci.mu.Unlock()

	currentTime := time.Now()

	for gateway, endTime := range ci.gatewayOutages {
		if endTime.Before(currentTime) {
			delete(ci.gatewayOutages, gateway)
		}
	}

	for component, endTime := range ci.networkPartitions {
		if endTime.Before(currentTime) {
			delete(ci.networkPartitions, component)
		}
	}
}

func (ci *ChaosInjector) EnableChaos() {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.activeChaos = true
	log.Println("Chaos injection enabled")
}

func (ci *ChaosInjector) DisableChaos() {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.activeChaos = false
	log.Println("Chaos injection disabled")
}

func (ci *ChaosInjector) SetIntensity(intensity ChaosIntensity) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	ci.config.Intensity = intensity
	ci.config.ApplyIntensityMultiplier()
	log.Printf("Chaos intensity set to %s", intensity)
}

func (ci *ChaosInjector) GetChaosMetrics() map[string]interface{} {
	baseMetrics := ci.monitor.GetMetrics()

	ci.mu.RLock()
	activeOutages := 0
	for _, endTime := range ci.gatewayOutages {
		if endTime.After(time.Now()) {
			activeOutages++
		}
	}

	activePartitions := 0
	for _, endTime := range ci.networkPartitions {
		if endTime.After(time.Now()) {
			activePartitions++
		}
	}

	enabledStrategies := make([]string, len(ci.config.EnabledStrategies))
	for i, strategy := range ci.config.EnabledStrategies {
		enabledStrategies[i] = string(strategy)
	}
	ci.mu.RUnlock()

	currentChaos := map[string]interface{}{
		"active_chaos":            ci.isChaosActive(),
		"intensity":               ci.config.Intensity,
		"active_gateway_outages":  activeOutages,
		"active_network_partitions": activePartitions,
		"success_rate_modifier":   ci.GetSuccessRateModifier(),
		"enabled_strategies":      enabledStrategies,
		"memory_usage":            ci.memoryLeakSimulator.GetMemoryUsage(),
		"active_cpu_spikes":       ci.cpuSpikeSimulator.GetActiveSpikes(),
	}

	for k, v := range currentChaos {
		baseMetrics[k] = v
	}
	return baseMetrics
}

type ChaosExperiment struct {
	injector    *ChaosInjector
	duration    time.Duration
	events      []ChaosEvent
	mu          sync.Mutex
}

func NewChaosExperiment(injector *ChaosInjector, duration time.Duration) *ChaosExperiment {
	return &ChaosExperiment{
		injector: injector,
		duration: duration,
		events:   make([]ChaosEvent, 0),
	}
}

func (ce *ChaosExperiment) Run() {
	log.Printf("Starting chaos experiment for %v", ce.duration)
	ce.injector.EnableChaos()

	startTime := time.Now()
	ticker := time.NewTicker(time.Duration(rand.Intn(20)+10) * time.Second)
	defer ticker.Stop()

	for time.Since(startTime) < ce.duration {
		select {
		case <-ticker.C:
			events := ce.injector.InjectPaymentChaos("experiment")
			ce.mu.Lock()
			ce.events = append(ce.events, events...)
			ce.mu.Unlock()
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}

	ce.injector.DisableChaos()
	ce.injector.memoryLeakSimulator.Cleanup()
	log.Printf("Chaos experiment completed. Injected %d events", len(ce.events))
}

func (ce *ChaosExperiment) GetEvents() []ChaosEvent {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	return ce.events
}

func CreateChaosInjector(intensity ChaosIntensity, enabledStrategies []ChaosStrategy) *ChaosInjector {
	if enabledStrategies == nil {
		enabledStrategies = []ChaosStrategy{Latency, Failure, GatewayOutage}
	}

	config := &ChaosConfig{
		Intensity:         intensity,
		EnabledStrategies: enabledStrategies,
	}
	config.ApplyIntensityMultiplier()

	return NewChaosInjector(config)
}

func demoChaosInjector() {
	rand.Seed(time.Now().UnixNano())

	injector := CreateChaosInjector(Medium, nil)

	fmt.Println("Chaos Injector Demo")
	fmt.Println("==================================================")

	experiment := NewChaosExperiment(injector, 10*time.Second)
	go experiment.Run()

	for i := 0; i < 5; i++ {
		events := injector.InjectPaymentChaos(fmt.Sprintf("demo_payment_%d", i))
		for _, event := range events {
			fmt.Printf("Chaos Event: %s\n", event.Description)
		}
		time.Sleep(1 * time.Second)
	}

	time.Sleep(2 * time.Second)

	metrics := injector.GetChaosMetrics()
	memoryUsage := injector.memoryLeakSimulator.GetMemoryUsage()

	metricsJSON, _ := json.MarshalIndent(metrics, "", "  ")
	memoryJSON, _ := json.MarshalIndent(memoryUsage, "", "  ")

	fmt.Printf("\nChaos Metrics: %s\n", string(metricsJSON))
	fmt.Printf("\nMemory Usage: %s\n", string(memoryJSON))

	injector.memoryLeakSimulator.Cleanup()
}

func main() {
	demoChaosInjector()
}
