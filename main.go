package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	_ "github.com/ClickHouse/clickhouse-go"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var fast_logger = log.New(os.Stderr, "", log.LstdFlags)

type BaselineExporterConfiguration struct {
	// Time before current time period for which we calculate baseline
	// 7 days by default, we use seconds
	CalculationPeriod int64 `json:"calculaton_period"`

	// Function used to find from traffic of all hosts in network: avg or max
	AggregationFunction string `json:"aggregation_function"`

	// Number of top talkers
	NumberOfTopTalkers uint64 `json:"number_of_top_talkers"`

	// info is default but can be debug
	LogLevel string `json:"log_level"`
}

// Configuration
type db_configuration_t struct {
	StorageBackend    string `json:"storage_backend"` // We support: mongodb, google_firestore, etcd
	Database_address  string `json:"mongodb_host"`
	Database_username string `json:"mongodb_username"`
	Database_password string `json:"mongodb_password"`
	Db_name           string `json:"mongodb_database_name"`
	Auth_source       string `json:"mongodb_auth_source"`
	Mongodb_port      uint   `json:"mongodb_port"`
}

type TrafficValue struct {
	Quantile95 int64 `bson:"quantile_95" json:"quantile_95"`
}

type TopTalker struct {
	// It can be IPv4 or IPv6
	Host  string `bson:"host" json:"host"`
	Value int64  `bson:"value" json:"value"`
}

type TrafficBaseline struct {
	Packets TrafficValue `bson:"packets" json:"packets"`
	Bits    TrafficValue `bson:"bits" json:"bits"`
	Flows   TrafficValue `bson:"flows" json:"flows"`

	Tcp_packets  TrafficValue `bson:"tcp_packets" json:"tcp_packets"`
	Udp_packets  TrafficValue `bson:"udp_packets" json:"udp_packets"`
	Icmp_packets TrafficValue `bson:"icmp_packets" json:"icmp_packets"`

	Fragmented_packets TrafficValue `bson:"fragmented_packets" json:"fragmented_packets"`
	Tcp_syn_packets    TrafficValue `bson:"tcp_syn_packets" json:"tcp_syn_packets"`

	Tcp_bits  TrafficValue `bson:"tcp_bits" json:"tcp_bits"`
	Udp_bits  TrafficValue `bson:"udp_bits" json:"udp_bits"`
	Icmp_bits TrafficValue `bson:"icmp_bits" json:"icmp_bits"`

	Fragmented_bits TrafficValue `bson:"fragmented_bits" json:"fragmented_bits"`
	Tcp_syn_bits    TrafficValue `bson:"tcp_syn_bits" json:"tcp_syn_bits"`
}

// Structure to push into MongoDB
type BaselineStructure struct {
	Name     string          `bson:"name" json:"name"`
	Incoming TrafficBaseline `bson:"incoming" json:"incoming" `
	Outgoing TrafficBaseline `bson:"outgoing" json:"outgoing"`
}

type AllTopTalkers struct {
	Packets []TopTalker `bson:"packets" json:"packets"`
	Bits    []TopTalker `bson:"bits" json:"bits"`
	Flows   []TopTalker `bson:"flows" json:"flows"`

	Tcp_packets  []TopTalker `bson:"tcp_packets" json:"tcp_packets" `
	Udp_packets  []TopTalker `bson:"udp_packets" json:"udp_packets"`
	Icmp_packets []TopTalker `bson:"icmp_packets" json:"icmp_packets"`

	Fragmented_packets []TopTalker `bson:"fragmented_packets" json:"fragmented_packets"`
	Tcp_syn_packets    []TopTalker `bson:"tcp_syn_packets" json:"tcp_syn_packets"`

	Tcp_bits  []TopTalker `bson:"tcp_bits" json:"tcp_bits"`
	Udp_bits  []TopTalker `bson:"udp_bits" json:"udp_bits"`
	Icmp_bits []TopTalker `bson:"icmp_bits" json:"icmp_bits"`

	Fragmented_bits []TopTalker `bson:"fragmented_bits" json:"fragmented_bits"`
	Tcp_syn_bits    []TopTalker `bson:"tcp_syn_bits" json:"tcp_syn_bits"`
}

// Structure to store top talkers in MongoDB
type TopTalkersStructure struct {
	Name     string        `bson:"name" json:"name"`
	Incoming AllTopTalkers `bson:"incoming" json:"incoming"`
	Outgoing AllTopTalkers `bson:"outgoing" json:"outgoing"`
}

var configuration BaselineExporterConfiguration

// Default path to tool configuration
var configuration_path = "/etc/fastnetmon/fastnetmon.conf"

var baseline_exporter_configuration_path = "/etc/fastnetmon/baseline_exporter.conf"

var log_path = "/var/log/fastnetmon/baseline_exporter.log"

// Default data to connect to MongoDB
var global_db_conf = db_configuration_t{Database_address: "127.0.0.1", Db_name: "fastnetmon", Database_username: "fastnetmon_user", Auth_source: "admin", Mongodb_port: 27017, StorageBackend: "mongodb"}

// This structure has only fields required for this app
type Fastnetmon_configuration_t struct {
	Clickhouse_metrics_database string `bson:"clickhouse_metrics_database" datastore:"clickhouse_metrics_database" json:"clickhouse_metrics_database" fastnetmon_type:"string" fastnetmon_description:"Database for ClickHouse traffic metrics" deprecated:"false" sensitive:"false"`
	Clickhouse_metrics_username string `bson:"clickhouse_metrics_username" datastore:"clickhouse_metrics_username" json:"clickhouse_metrics_username" fastnetmon_type:"string" fastnetmon_description:"Username for ClickHouse metrics" deprecated:"false" sensitive:"false"`
	Clickhouse_metrics_password string `bson:"clickhouse_metrics_password" datastore:"clickhouse_metrics_password" json:"clickhouse_metrics_password" fastnetmon_type:"string" fastnetmon_description:"Password for ClickHouse metrics" deprecated:"false" sensitive:"false"`
	Clickhouse_metrics_host     string `bson:"clickhouse_metrics_host" datastore:"clickhouse_metrics_host" json:"clickhouse_metrics_host" fastnetmon_type:"numeric_ipv4_host" fastnetmon_description:"Server address for ClickHouse metric" deprecated:"false" sensitive:"false"`
	Clickhouse_metrics_port     uint   `bson:"clickhouse_metrics_port" datastore:"clickhouse_metrics_port" json:"clickhouse_metrics_port" fastnetmon_type:"numeric_ipv4_port" fastnetmon_description:"ClickHouse server port" deprecated:"false" sensitive:"false"`
}

// This structure has only fields required for this app
type Ban_settings_t struct {
	Name               string   `bson:"name" datastore:"name" json:"name" fastnetmon_type:"string" fastnetmon_description:"Name of host group" deprecated:"false" sensitive:"false"`
	Networks           []string `bson:"networks" datastore:"networks" json:"networks" fastnetmon_type:"cidr_networks_list" fastnetmon_description:"List of networks which belong to this group" deprecated:"false" sensitive:"false"`
	Calculation_method string   `bson:"calculation_method" datastore:"calculation_method" json:"calculation_method" fastnetmon_type:"string" fastnetmon_description:"Traffic calculation method for host group: total or per_host (or empty value)" deprecated:"false" sensitive:"false"`
}

// Mongo
var (
	current_global_conf Fastnetmon_configuration_t
)

func main() {
	if os.Geteuid() != 0 || os.Getegid() != 0 {
		log.Fatal("Please run this tool with root rights (e.g. with sudo)")
	}

	log_file, err := os.OpenFile(log_path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		log.Fatalf("Cannot open log file: %v", err)
	}

	defer log_file.Close()

	multi_writer := io.MultiWriter(os.Stdout, log_file)

	fast_logger.SetOutput(multi_writer)

	fast_logger.Print("Started Baseline exporter")

	// Calculate data over last 7 days by default
	configuration.CalculationPeriod = 7 * 24 * 3600
	configuration.AggregationFunction = "quantile(0.95)"
	configuration.NumberOfTopTalkers = 100
	configuration.LogLevel = "info"

	if is_file_exists(baseline_exporter_configuration_path) {

		file_as_array, err := ioutil.ReadFile(baseline_exporter_configuration_path)

		if err != nil {
			fast_logger.Fatalf("Could not read configuration file %s with error: %v", baseline_exporter_configuration_path, err)
		}

		// This command will override our default configuration
		err = json.Unmarshal(file_as_array, &configuration)

		if err != nil {
			fast_logger.Fatalf("Could not read JSON configuration: %v", err)
		}

		fast_logger.Printf("Successfully read configuration file: %+v", configuration)

	} else {
		fast_logger.Printf("We have no configuration file %s, start with default options", baseline_exporter_configuration_path)
	}

	fast_logger.Printf("Baseline exporter configuration: %+v", configuration)

	// If we have custom file with configuration for MongoDB
	if is_file_exists(configuration_path) {
		file_as_array, err := ioutil.ReadFile(configuration_path)

		if err != nil {
			fast_logger.Fatal(fmt.Sprintf("Could not read configuration file from %s with error: %v", configuration_path, err))
		}

		// This command will override our default MongoDB configuration
		err = json.Unmarshal(file_as_array, &global_db_conf)

		if err != nil {
			fast_logger.Fatalf("Could not read json configuration: %v", err)
		}

		log.Printf("Read custom database configuration from %s", configuration_path)
	}

	fastnetmon_password_binary, _ := ioutil.ReadFile("/etc/fastnetmon/keychain/.mongo_fastnetmon_password")

	// Read password from file
	global_db_conf.Database_password = string(fastnetmon_password_binary)

	mongodb_address := global_db_conf.Database_address

	// Add brackets for IPv6 addresses
	if strings.Contains(mongodb_address, ":") {
		mongodb_address = "[" + mongodb_address + "]"
	}

	mongodb_full_address := fmt.Sprintf("%s:%d", mongodb_address, global_db_conf.Mongodb_port)

	// Estasblish connection to MongoDB usin new driver
	mongo_uri := "mongodb://" + global_db_conf.Database_username + ":" + global_db_conf.Database_password + "@" + mongodb_full_address

	// Create a new client and connect to the server
	mongo_client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongo_uri))
	if err != nil {
		fast_logger.Fatalf("Cannot establish connection to MongoDB: %v", err)
	}

	defer func() {
		if err = mongo_client.Disconnect(context.TODO()); err != nil {
			fast_logger.Fatalf("Cannot disconnect from MongoDB: %v", err)
		}
	}()

	// Ping the primary
	if err := mongo_client.Ping(context.TODO(), readpref.Primary()); err != nil {
		fast_logger.Fatalf("Cannot PING MongoDB: %v", err)
	}

	fast_logger.Printf("Successfully connected to MongoDB and executed PING query successfully")

	// Read main configuration
	main_collection := mongo_client.Database(global_db_conf.Db_name).Collection("configuration")

	err = main_collection.FindOne(context.TODO(), bson.D{}).Decode(&current_global_conf)

	if err != nil {
		fast_logger.Fatalf("Could not retrieve main configuration from MongoDB: %v", err)
	}

	fast_logger.Printf("Successfully read main configuration of FastNetMon from MongoDB")

	// Read all hostgroups
	fast_logger.Printf("Preparing to read all hostgroups")

	hostgroups_collection := mongo_client.Database(global_db_conf.Db_name).Collection("hostgroups_configuration")

	var host_groups []Ban_settings_t

	cursor, err := hostgroups_collection.Find(context.TODO(), bson.D{})

	if err != nil {
		fast_logger.Fatalf("Cannot load hostgroups from MongoDB: %v", err)
	}

	if err = cursor.All(context.Background(), &host_groups); err != nil {
		fast_logger.Fatalf("Cannot retrieve hostgroups from MongoDB: %v", err)
	}

	if len(host_groups) == 0 {
		fast_logger.Fatalf("We do not have host groups for your query")
	}

	fast_logger.Printf("Loaded %d hostgroups", len(host_groups))

	for _, host_group := range host_groups {
		fast_logger.Printf("Hostgroup %s loaded with networks %v", host_group.Name, strings.Join(host_group.Networks, ","))
	}

	// fast_logger.Printf("%+v", current_global_conf)

	// fast_logger.Printf("Read custom database configuration: %+v", configuration)

	log.Printf("Trying to connect to Clickhouse on %s:%d", current_global_conf.Clickhouse_metrics_host, current_global_conf.Clickhouse_metrics_port)

	// You can add: ?debug=true for debugging
	clickhouse_client, err := sql.Open("clickhouse", fmt.Sprintf("tcp://%s:%d", current_global_conf.Clickhouse_metrics_host, current_global_conf.Clickhouse_metrics_port))

	if err != nil {
		fast_logger.Fatalf("Cannot connect to Clickhouse: %v", err)
	}

	if err := clickhouse_client.Ping(); err != nil {
		fast_logger.Fatalf("Cannot connect to Clickhouse: %v", err)
	}

	fast_logger.Printf("Successfully connected to Clickhouse")

	for _, host_group := range host_groups {
		// We do processing only for per_host hostgroups
		if host_group.Calculation_method == "total" {
			continue
		}

		fast_logger.Printf("Start baseline generation for %s", host_group.Name)

		metrics, err := generate_baselines(host_group.Name, host_group.Networks, clickhouse_client, configuration.AggregationFunction)

		if err != nil {
			// OK, we can tolerate some failures
			fast_logger.Printf("Cannot generate baselines for %s with error %v", host_group.Name, err)
			continue
		}

		hostgroups_baseline_collection := mongo_client.Database(global_db_conf.Db_name).Collection("baseline_exporter_hostgroups_baseline")

		filter := bson.D{{"name", host_group.Name}}

		true_bool := new(bool)
		*true_bool = true

		_, err = hostgroups_baseline_collection.ReplaceOne(context.TODO(), filter, metrics, &options.ReplaceOptions{Upsert: true_bool})

		if err != nil {
			fast_logger.Printf("Cannot update baseline for %s in MongoDB: %v", host_group.Name, err)
			continue
		}

		fast_logger.Printf("Updated baseline in MongoDB for %s", host_group.Name)

		if configuration.LogLevel == "debug" {
			fast_logger.Printf("Metrics: %+v", metrics)
		}
	}

	// We have another loop to generate top talkers

	for _, host_group := range host_groups {
		// We do processing only for per_host hostgroups
		if host_group.Calculation_method == "total" {
			continue
		}

		fast_logger.Printf("Start top talkers generation for %s", host_group.Name)

		top_talkers, err := get_top_talkers_by_all_fields(host_group.Name, host_group.Networks, clickhouse_client, configuration.NumberOfTopTalkers)

		if err != nil {
			fast_logger.Printf("Cannot get top talkers for %s with error %v", host_group.Name, err)
			continue
		}

		if configuration.LogLevel == "debug" {
			fast_logger.Printf("Top talkers: %+v", top_talkers)
		}

		hostgroups_top_talkers_collection := mongo_client.Database(global_db_conf.Db_name).Collection("baseline_exporter_hostgroups_top_talkers")

		filter := bson.D{{"name", host_group.Name}}

		true_bool := new(bool)
		*true_bool = true

		_, err = hostgroups_top_talkers_collection.ReplaceOne(context.TODO(), filter, top_talkers, &options.ReplaceOptions{Upsert: true_bool})

		if err != nil {
			fast_logger.Printf("Cannot update top talkers for %s in MongoDB: %v", host_group.Name, err)
			continue
		}

		fast_logger.Printf("Updated top talkers in MongoDB for %s", host_group.Name)
	}
}

// Generates network WHERE clause to lookup IP in many IPv4 and IPv6 networks
func generate_network_where_clause(networks_list []string) string {

	all_sql_query_where_clauses := []string{}

	for _, network_string := range networks_list {
		_, _, err := net.ParseCIDR(network_string)

		if err != nil {
			fast_logger.Printf("Format error for prefix %s: %v", network_string, err)
			continue
		}

		query_in_clause := fmt.Sprintf("isIPAddressInRange(host, '%s')", network_string)

		all_sql_query_where_clauses = append(all_sql_query_where_clauses, query_in_clause)
	}

	// We handle it that way for global group which has no networks at all
	merged_where_clause_by_networks := "1 = 1"

	if len(all_sql_query_where_clauses) > 0 {
		merged_where_clause_by_networks = strings.Join(all_sql_query_where_clauses, " OR ")
	}

	return merged_where_clause_by_networks
}

// Returns WHERE section to filter by date and date time
func generate_date_filter() string {
	return fmt.Sprintf("metricDate >= toDate(now() - %d) and (metricDateTime >= now() - %d)", configuration.CalculationPeriod, configuration.CalculationPeriod)
}

// Creates number of top talkers by using all possible metrics
func get_top_talkers_by_all_fields(hostgroup_name string, networks_list []string, clickhouse_client *sql.DB, top_talkers_number uint64) (*TopTalkersStructure, error) {
	all_top_talkers := TopTalkersStructure{}

	all_top_talkers.Name = hostgroup_name

	fields_for_processing := map[string]*[]TopTalker{
		"packets_incoming": &all_top_talkers.Incoming.Packets,
		"packets_outgoing": &all_top_talkers.Outgoing.Packets,
		"bits_incoming":    &all_top_talkers.Incoming.Bits,
		"bits_outgoing":    &all_top_talkers.Outgoing.Bits,
		"flows_incoming":   &all_top_talkers.Incoming.Flows,
		"flows_outgoing":   &all_top_talkers.Outgoing.Flows,

		// Per protocol counters
		"tcp_packets_incoming":        &all_top_talkers.Incoming.Tcp_packets,
		"tcp_packets_outgoing":        &all_top_talkers.Outgoing.Tcp_packets,
		"udp_packets_incoming":        &all_top_talkers.Incoming.Udp_packets,
		"udp_packets_outgoing":        &all_top_talkers.Outgoing.Udp_packets,
		"icmp_packets_incoming":       &all_top_talkers.Incoming.Icmp_packets,
		"icmp_packets_outgoing":       &all_top_talkers.Outgoing.Icmp_packets,
		"fragmented_packets_incoming": &all_top_talkers.Incoming.Fragmented_packets,
		"fragmented_packets_outgoing": &all_top_talkers.Outgoing.Fragmented_packets,
		"tcp_syn_packets_incoming":    &all_top_talkers.Incoming.Tcp_syn_packets,
		"tcp_syn_packets_outgoing":    &all_top_talkers.Outgoing.Tcp_syn_packets,

		"tcp_bits_incoming":        &all_top_talkers.Incoming.Tcp_bits,
		"tcp_bits_outgoing":        &all_top_talkers.Outgoing.Tcp_bits,
		"udp_bits_incoming":        &all_top_talkers.Incoming.Udp_bits,
		"udp_bits_outgoing":        &all_top_talkers.Outgoing.Udp_bits,
		"icmp_bits_incoming":       &all_top_talkers.Incoming.Icmp_bits,
		"icmp_bits_outgoing":       &all_top_talkers.Outgoing.Icmp_bits,
		"fragmented_bits_incoming": &all_top_talkers.Incoming.Fragmented_bits,
		"fragmented_bits_outgoing": &all_top_talkers.Outgoing.Fragmented_bits,
		"tcp_syn_bits_incoming":    &all_top_talkers.Incoming.Tcp_syn_bits,
		"tcp_syn_bits_outgoing":    &all_top_talkers.Outgoing.Tcp_syn_bits,
	}

	for metric_type, target_field := range fields_for_processing {
		top_talkers, err := get_top_talkers_by_field(hostgroup_name, networks_list, clickhouse_client, metric_type, top_talkers_number)

		if err != nil {
			return nil, fmt.Errorf("Cannot generate top talkers by field %s with error %v", metric_type, err)
		}

		*target_field = top_talkers

		// fast_logger.Printf("Top talkers by %s are %+v", metric_type, top_talkers)
	}

	// fast_logger.Printf("Top talkers: %+v", all_top_talkers)
	return &all_top_talkers, nil
}

// Get top talkers ordered by specific type of traffic passed in field_for_query
func get_top_talkers_by_field(hostgroup_name string, networks_list []string, clickhouse_client *sql.DB, field_for_query string, top_talkers_number uint64) ([]TopTalker, error) {
	merged_where_clause_by_networks := generate_network_where_clause(networks_list)

	// We use max to aggregate top talkers
	aggregation_function := "max"

	query := fmt.Sprintf("SELECT host, %s(toInt64(%s)) as max_value FROM %s.%s WHERE (%s) AND (%s) GROUP by host ORDER BY max_value DESC LIMIT %d", aggregation_function, field_for_query, current_global_conf.Clickhouse_metrics_database, "host_metrics", generate_date_filter(), merged_where_clause_by_networks, top_talkers_number)

	if configuration.LogLevel == "debug" {
		fast_logger.Printf("SQL Query: %s\n", query)
	}

	rows, err := clickhouse_client.Query(query)

	if err != nil {
		return nil, fmt.Errorf("Cannot execute Clickhouse query '%s' with error: %w\n", query, err)
	}

	top_talkers := []TopTalker{}

	for rows.Next() {
		var top_talker TopTalker

		err := rows.Scan(&top_talker.Host, &top_talker.Value)

		if err != nil {
			return nil, errors.Errorf("Cannot read row: %v", err)
		}

		top_talkers = append(top_talkers, top_talker)
	}

	return top_talkers, nil
}

// Generates baseline for list of networks according to Clickhosue history data
func generate_baselines(hostgroup_name string, networks_list []string, clickhouse_client *sql.DB, aggregation_function string) (*BaselineStructure, error) {
	merged_where_clause_by_networks := generate_network_where_clause(networks_list)

	fields_for_processing := []string{
		"packets_incoming",
		"packets_outgoing",
		"bits_incoming",
		"bits_outgoing",
		"flows_incoming",
		"flows_outgoing",

		// Per protocol counters
		"tcp_packets_incoming",
		"tcp_packets_outgoing",
		"udp_packets_incoming",
		"udp_packets_outgoing",
		"icmp_packets_incoming",
		"icmp_packets_outgoing",
		"fragmented_packets_incoming",
		"fragmented_packets_outgoing",
		"tcp_syn_packets_incoming",
		"tcp_syn_packets_outgoing",
		"tcp_bits_incoming",
		"tcp_bits_outgoing",
		"udp_bits_incoming",
		"udp_bits_outgoing",
		"icmp_bits_incoming",
		"icmp_bits_outgoing",
		"fragmented_bits_incoming",
		"fragmented_bits_outgoing",
		"tcp_syn_bits_incoming",
		"tcp_syn_bits_outgoing",
	}

	fields_for_processing = processMap(fields_for_processing, func(value string) string {
		return fmt.Sprintf("toInt64(%s(%s))", aggregation_function, value)
	})

	query := fmt.Sprintf("SELECT COUNT(*), %s FROM %s.%s WHERE (%s) AND (%s)", strings.Join(fields_for_processing, ","), current_global_conf.Clickhouse_metrics_database, "host_metrics", generate_date_filter(), merged_where_clause_by_networks)

	if configuration.LogLevel == "debug" {
		fast_logger.Printf("SQL Query: %s\n", query)
	}

	rows, err := clickhouse_client.Query(query)

	if err != nil {
		return nil, fmt.Errorf("Cannot execute Clickhouse query '%s' with error: %w\n", query, err)
	}

	fast_logger.Printf("Retrieve traffic metrics for hostgroup %s", hostgroup_name)

	for rows.Next() {
		metrics_row := &BaselineStructure{}
		metrics_row.Name = hostgroup_name

		var hosts_with_traffic int64

		err := rows.Scan(&hosts_with_traffic,
			&metrics_row.Incoming.Packets.Quantile95,
			&metrics_row.Outgoing.Packets.Quantile95,
			&metrics_row.Incoming.Bits.Quantile95,
			&metrics_row.Outgoing.Bits.Quantile95,
			&metrics_row.Incoming.Flows.Quantile95,
			&metrics_row.Outgoing.Flows.Quantile95,
			&metrics_row.Incoming.Tcp_packets.Quantile95,
			&metrics_row.Outgoing.Tcp_packets.Quantile95,
			&metrics_row.Incoming.Udp_packets.Quantile95,
			&metrics_row.Outgoing.Udp_packets.Quantile95,
			&metrics_row.Incoming.Icmp_packets.Quantile95,
			&metrics_row.Outgoing.Icmp_packets.Quantile95,
			&metrics_row.Incoming.Fragmented_packets.Quantile95,
			&metrics_row.Outgoing.Fragmented_packets.Quantile95,
			&metrics_row.Incoming.Tcp_syn_packets.Quantile95,
			&metrics_row.Outgoing.Tcp_syn_packets.Quantile95,
			&metrics_row.Incoming.Tcp_bits.Quantile95,
			&metrics_row.Outgoing.Tcp_bits.Quantile95,
			&metrics_row.Incoming.Udp_bits.Quantile95,
			&metrics_row.Outgoing.Udp_bits.Quantile95,
			&metrics_row.Incoming.Icmp_bits.Quantile95,
			&metrics_row.Outgoing.Icmp_bits.Quantile95,
			&metrics_row.Incoming.Fragmented_bits.Quantile95,
			&metrics_row.Outgoing.Fragmented_bits.Quantile95,
			&metrics_row.Incoming.Tcp_syn_bits.Quantile95,
			&metrics_row.Outgoing.Tcp_syn_bits.Quantile95)

		if err != nil {
			return nil, errors.Errorf("Cannot read row: %v", err)
		}

		return metrics_row, nil
	}

	return nil, fmt.Errorf("There are no data in CLickhouse")
}

// Applies function to all elements
func processMap(incoming []string, f func(string) string) []string {
	outgoing := make([]string, len(incoming))

	for index, value := range incoming {
		outgoing[index] = f(value)
	}

	return outgoing
}

func cast_to_int_64(value interface{}) int64 {
	switch value.(type) {
	case float64:
		return int64(value.(float64))
	case int64:
		return value.(int64)
	}

	return 0
}

func cast_to_uint(value interface{}) uint {
	switch value.(type) {
	case float64:
		return uint(value.(float64))
	case uint:
		return value.(uint)
	}

	return 0
}

/* Is this file exists? */
func is_file_exists(file_path string) bool {
	if _, err := os.Stat(file_path); os.IsNotExist(err) {
		return false
	}

	return true
}
