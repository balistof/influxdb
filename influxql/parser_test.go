package influxql_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/influxdb/influxdb/influxql"
)

// Ensure the parser can parse a multi-statement query.
func TestParser_ParseQuery(t *testing.T) {
	s := `SELECT a FROM b; SELECT c FROM d`
	q, err := influxql.NewParser(strings.NewReader(s)).ParseQuery()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	} else if len(q.Statements) != 2 {
		t.Fatalf("unexpected statement count: %d", len(q.Statements))
	}
}

// Ensure the parser can parse an empty query.
func TestParser_ParseQuery_Empty(t *testing.T) {
	q, err := influxql.NewParser(strings.NewReader(``)).ParseQuery()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	} else if len(q.Statements) != 0 {
		t.Fatalf("unexpected statement count: %d", len(q.Statements))
	}
}

// Ensure the parser can return an error from an malformed statement.
func TestParser_ParseQuery_ParseError(t *testing.T) {
	_, err := influxql.NewParser(strings.NewReader(`SELECT`)).ParseQuery()
	if err == nil || err.Error() != `found EOF, expected identifier, string, number, bool at line 1, char 8` {
		t.Fatalf("unexpected error: %s", err)
	}
}

// Ensure the parser can parse strings into Statement ASTs.
func TestParser_ParseStatement(t *testing.T) {
	var tests = []struct {
		s    string
		stmt influxql.Statement
		err  string
	}{
		// SELECT * statement
		{
			s: `SELECT * FROM myseries`,
			stmt: &influxql.SelectStatement{
				Fields: []*influxql.Field{
					{Expr: &influxql.Wildcard{}},
				},
				Source: &influxql.Measurement{Name: "myseries"},
			},
		},

		// SELECT statement
		{
			s: `SELECT field1, field2 ,field3 AS field_x FROM myseries WHERE host = 'hosta.influxdb.org' GROUP BY 10h ORDER BY ASC LIMIT 20 OFFSET 10;`,
			stmt: &influxql.SelectStatement{
				Fields: []*influxql.Field{
					{Expr: &influxql.VarRef{Val: "field1"}},
					{Expr: &influxql.VarRef{Val: "field2"}},
					{Expr: &influxql.VarRef{Val: "field3"}, Alias: "field_x"},
				},
				Source: &influxql.Measurement{Name: "myseries"},
				Condition: &influxql.BinaryExpr{
					Op:  influxql.EQ,
					LHS: &influxql.VarRef{Val: "host"},
					RHS: &influxql.StringLiteral{Val: "hosta.influxdb.org"},
				},
				Dimensions: []*influxql.Dimension{
					{Expr: &influxql.DurationLiteral{Val: 10 * time.Hour}},
				},
				SortFields: []*influxql.SortField{
					{Ascending: true},
				},
				Limit:  20,
				Offset: 10,
			},
		},

		// SELECT statement with JOIN
		{
			s: `SELECT field1 FROM join(aa,"bb", cc) JOIN cc`,
			stmt: &influxql.SelectStatement{
				Fields: []*influxql.Field{{Expr: &influxql.VarRef{Val: "field1"}}},
				Source: &influxql.Join{
					Measurements: []*influxql.Measurement{
						{Name: "aa"},
						{Name: `"bb"`},
						{Name: "cc"},
					},
				},
			},
		},

		// SELECT statement with MERGE
		{
			s: `SELECT field1 FROM merge(aa,b.b)`,
			stmt: &influxql.SelectStatement{
				Fields: []*influxql.Field{{Expr: &influxql.VarRef{Val: "field1"}}},
				Source: &influxql.Merge{
					Measurements: []*influxql.Measurement{
						{Name: "aa"},
						{Name: "b.b"},
					},
				},
			},
		},

		// SELECT statement (lowercase)
		{
			s: `select my_field from myseries`,
			stmt: &influxql.SelectStatement{
				Fields: []*influxql.Field{{Expr: &influxql.VarRef{Val: "my_field"}}},
				Source: &influxql.Measurement{Name: "myseries"},
			},
		},

		// SELECT statement with multiple ORDER BY fields
		{
			s: `SELECT field1 FROM myseries ORDER BY ASC, field1, field2 DESC LIMIT 10`,
			stmt: &influxql.SelectStatement{
				Fields: []*influxql.Field{{Expr: &influxql.VarRef{Val: "field1"}}},
				Source: &influxql.Measurement{Name: "myseries"},
				SortFields: []*influxql.SortField{
					{Ascending: true},
					{Name: "field1"},
					{Name: "field2"},
				},
				Limit: 10,
			},
		},

		// DELETE statement
		{
			s: `DELETE FROM myseries WHERE host = 'hosta.influxdb.org'`,
			stmt: &influxql.DeleteStatement{
				Source: &influxql.Measurement{Name: "myseries"},
				Condition: &influxql.BinaryExpr{
					Op:  influxql.EQ,
					LHS: &influxql.VarRef{Val: "host"},
					RHS: &influxql.StringLiteral{Val: "hosta.influxdb.org"},
				},
			},
		},

		// SHOW DATABASES
		{
			s:    `SHOW DATABASES`,
			stmt: &influxql.ShowDatabasesStatement{},
		},

		// SHOW SERIES statement
		{
			s:    `SHOW SERIES`,
			stmt: &influxql.ShowSeriesStatement{},
		},

		// SHOW SERIES WHERE with ORDER BY and LIMIT
		{
			s: `SHOW SERIES WHERE region = 'uswest' ORDER BY ASC, field1, field2 DESC LIMIT 10`,
			stmt: &influxql.ShowSeriesStatement{
				Condition: &influxql.BinaryExpr{
					Op:  influxql.EQ,
					LHS: &influxql.VarRef{Val: "region"},
					RHS: &influxql.StringLiteral{Val: "uswest"},
				},
				SortFields: []*influxql.SortField{
					{Ascending: true},
					{Name: "field1"},
					{Name: "field2"},
				},
				Limit: 10,
			},
		},

		// SHOW MEASUREMENTS WHERE with ORDER BY and LIMIT
		{
			s: `SHOW MEASUREMENTS WHERE region = 'uswest' ORDER BY ASC, field1, field2 DESC LIMIT 10`,
			stmt: &influxql.ShowMeasurementsStatement{
				Condition: &influxql.BinaryExpr{
					Op:  influxql.EQ,
					LHS: &influxql.VarRef{Val: "region"},
					RHS: &influxql.StringLiteral{Val: "uswest"},
				},
				SortFields: []*influxql.SortField{
					{Ascending: true},
					{Name: "field1"},
					{Name: "field2"},
				},
				Limit: 10,
			},
		},

		// SHOW RETENTION POLICIES
		{
			s: `SHOW RETENTION POLICIES mydb`,
			stmt: &influxql.ShowRetentionPoliciesStatement{
				Database: "mydb",
			},
		},

		// SHOW TAG KEYS
		{
			s: `SHOW TAG KEYS FROM src`,
			stmt: &influxql.ShowTagKeysStatement{
				Source: &influxql.Measurement{Name: "src"},
			},
		},

		// SHOW TAG KEYS
		{
			s: `SHOW TAG KEYS FROM src WHERE region = 'uswest' ORDER BY ASC, field1, field2 DESC LIMIT 10`,
			stmt: &influxql.ShowTagKeysStatement{
				Source: &influxql.Measurement{Name: "src"},
				Condition: &influxql.BinaryExpr{
					Op:  influxql.EQ,
					LHS: &influxql.VarRef{Val: "region"},
					RHS: &influxql.StringLiteral{Val: "uswest"},
				},
				SortFields: []*influxql.SortField{
					{Ascending: true},
					{Name: "field1"},
					{Name: "field2"},
				},
				Limit: 10,
			},
		},

		// SHOW TAG VALUES FROM ... WITH KEY = ...
		{
			s: `SHOW TAG VALUES FROM src WITH KEY = region WHERE region = 'uswest' ORDER BY ASC, field1, field2 DESC LIMIT 10`,
			stmt: &influxql.ShowTagValuesStatement{
				Source:  &influxql.Measurement{Name: "src"},
				TagKeys: []string{"region"},
				Condition: &influxql.BinaryExpr{
					Op:  influxql.EQ,
					LHS: &influxql.VarRef{Val: "region"},
					RHS: &influxql.StringLiteral{Val: "uswest"},
				},
				SortFields: []*influxql.SortField{
					{Ascending: true},
					{Name: "field1"},
					{Name: "field2"},
				},
				Limit: 10,
			},
		},

		// SHOW TAG VALUES FROM ... WITH KEY IN...
		{
			s: `SHOW TAG VALUES FROM cpu WITH KEY IN (region, host) WHERE region = 'uswest'`,
			stmt: &influxql.ShowTagValuesStatement{
				Source:  &influxql.Measurement{Name: "cpu"},
				TagKeys: []string{"region", "host"},
				Condition: &influxql.BinaryExpr{
					Op:  influxql.EQ,
					LHS: &influxql.VarRef{Val: "region"},
					RHS: &influxql.StringLiteral{Val: "uswest"},
				},
			},
		},

		// SHOW TAG VALUES ... AND TAG KEY =
		{
			s: `SHOW TAG VALUES FROM cpu WITH KEY IN (region,service,host)WHERE region = 'uswest'`,
			stmt: &influxql.ShowTagValuesStatement{
				Source:  &influxql.Measurement{Name: "cpu"},
				TagKeys: []string{"region", "service", "host"},
				Condition: &influxql.BinaryExpr{
					Op:  influxql.EQ,
					LHS: &influxql.VarRef{Val: "region"},
					RHS: &influxql.StringLiteral{Val: "uswest"},
				},
			},
		},

		// SHOW TAG VALUES WITH KEY = ...
		{
			s: `SHOW TAG VALUES WITH KEY = host WHERE region = 'uswest'`,
			stmt: &influxql.ShowTagValuesStatement{
				TagKeys: []string{"host"},
				Condition: &influxql.BinaryExpr{
					Op:  influxql.EQ,
					LHS: &influxql.VarRef{Val: "region"},
					RHS: &influxql.StringLiteral{Val: "uswest"},
				},
			},
		},

		// SHOW USERS
		{
			s:    `SHOW USERS`,
			stmt: &influxql.ShowUsersStatement{},
		},

		// SHOW FIELD KEYS
		{
			s: `SHOW FIELD KEYS FROM src WHERE region = 'uswest' ORDER BY ASC, field1, field2 DESC LIMIT 10`,
			stmt: &influxql.ShowFieldKeysStatement{
				Source: &influxql.Measurement{Name: "src"},
				Condition: &influxql.BinaryExpr{
					Op:  influxql.EQ,
					LHS: &influxql.VarRef{Val: "region"},
					RHS: &influxql.StringLiteral{Val: "uswest"},
				},
				SortFields: []*influxql.SortField{
					{Ascending: true},
					{Name: "field1"},
					{Name: "field2"},
				},
				Limit: 10,
			},
		},

		// DROP SERIES statement
		{
			s:    `DROP SERIES myseries`,
			stmt: &influxql.DropSeriesStatement{Name: "myseries"},
		},

		// SHOW CONTINUOUS QUERIES statement
		{
			s:    `SHOW CONTINUOUS QUERIES`,
			stmt: &influxql.ShowContinuousQueriesStatement{},
		},

		// CREATE CONTINUOUS QUERY ... INTO <measurement>
		{
			s: `CREATE CONTINUOUS QUERY myquery ON testdb BEGIN SELECT count() INTO measure1 FROM myseries GROUP BY time(5m) END`,
			stmt: &influxql.CreateContinuousQueryStatement{
				Name:     "myquery",
				Database: "testdb",
				Source: &influxql.SelectStatement{
					Fields: []*influxql.Field{{Expr: &influxql.Call{Name: "count"}}},
					Target: &influxql.Target{Measurement: "measure1"},
					Source: &influxql.Measurement{Name: "myseries"},
					Dimensions: []*influxql.Dimension{
						&influxql.Dimension{
							Expr: &influxql.Call{
								Name: "time",
								Args: []influxql.Expr{
									&influxql.DurationLiteral{Val: 5 * time.Minute},
								},
							},
						},
					},
				},
			},
		},

		// CREATE CONTINUOUS QUERY ... INTO <retention-policy>.<measurement>
		{
			s: `CREATE CONTINUOUS QUERY myquery ON testdb BEGIN SELECT count() INTO "1h.policy1"."cpu.load" FROM myseries GROUP BY time(5m) END`,
			stmt: &influxql.CreateContinuousQueryStatement{
				Name:     "myquery",
				Database: "testdb",
				Source: &influxql.SelectStatement{
					Fields: []*influxql.Field{{Expr: &influxql.Call{Name: "count"}}},
					Target: &influxql.Target{
						Measurement: `"1h.policy1"."cpu.load"`,
					},
					Source: &influxql.Measurement{Name: "myseries"},
					Dimensions: []*influxql.Dimension{
						&influxql.Dimension{
							Expr: &influxql.Call{
								Name: "time",
								Args: []influxql.Expr{
									&influxql.DurationLiteral{Val: 5 * time.Minute},
								},
							},
						},
					},
				},
			},
		},

		// CREATE DATABASE statement
		{
			s: `CREATE DATABASE testdb`,
			stmt: &influxql.CreateDatabaseStatement{
				Name: "testdb",
			},
		},

		// CREATE USER statement
		{
			s: `CREATE USER testuser WITH PASSWORD 'pwd1337'`,
			stmt: &influxql.CreateUserStatement{
				Name:     "testuser",
				Password: "pwd1337",
			},
		},

		// CREATE USER ... WITH ALL PRIVILEGES
		{
			s: `CREATE USER testuser WITH PASSWORD 'pwd1337' WITH ALL PRIVILEGES`,
			stmt: &influxql.CreateUserStatement{
				Name:      "testuser",
				Password:  "pwd1337",
				Privilege: influxql.NewPrivilege(influxql.AllPrivileges),
			},
		},

		// DROP CONTINUOUS QUERY statement
		{
			s:    `DROP CONTINUOUS QUERY myquery`,
			stmt: &influxql.DropContinuousQueryStatement{Name: "myquery"},
		},

		// DROP DATABASE statement
		{
			s:    `DROP DATABASE testdb`,
			stmt: &influxql.DropDatabaseStatement{Name: "testdb"},
		},

		// DROP RETENTION POLICY
		{
			s: `DROP RETENTION POLICY "1h.cpu" ON mydb`,
			stmt: &influxql.DropRetentionPolicyStatement{
				Name:     `"1h.cpu"`,
				Database: `mydb`,
			},
		},

		// DROP USER statement
		{
			s:    `DROP USER jdoe`,
			stmt: &influxql.DropUserStatement{Name: "jdoe"},
		},

		// GRANT READ
		{
			s: `GRANT READ ON testdb TO jdoe`,
			stmt: &influxql.GrantStatement{
				Privilege: influxql.ReadPrivilege,
				On:        "testdb",
				User:      "jdoe",
			},
		},

		// GRANT WRITE
		{
			s: `GRANT WRITE ON testdb TO jdoe`,
			stmt: &influxql.GrantStatement{
				Privilege: influxql.WritePrivilege,
				On:        "testdb",
				User:      "jdoe",
			},
		},

		// GRANT ALL
		{
			s: `GRANT ALL ON testdb TO jdoe`,
			stmt: &influxql.GrantStatement{
				Privilege: influxql.AllPrivileges,
				On:        "testdb",
				User:      "jdoe",
			},
		},

		// GRANT ALL PRIVILEGES
		{
			s: `GRANT ALL PRIVILEGES ON testdb TO jdoe`,
			stmt: &influxql.GrantStatement{
				Privilege: influxql.AllPrivileges,
				On:        "testdb",
				User:      "jdoe",
			},
		},

		// GRANT cluster admin
		{
			s: `GRANT ALL PRIVILEGES TO jdoe`,
			stmt: &influxql.GrantStatement{
				Privilege: influxql.AllPrivileges,
				User:      "jdoe",
			},
		},

		// REVOKE READ
		{
			s: `REVOKE READ on testdb FROM jdoe`,
			stmt: &influxql.RevokeStatement{
				Privilege: influxql.ReadPrivilege,
				On:        "testdb",
				User:      "jdoe",
			},
		},

		// REVOKE WRITE
		{
			s: `REVOKE WRITE ON testdb FROM jdoe`,
			stmt: &influxql.RevokeStatement{
				Privilege: influxql.WritePrivilege,
				On:        "testdb",
				User:      "jdoe",
			},
		},

		// REVOKE ALL
		{
			s: `REVOKE ALL ON testdb FROM jdoe`,
			stmt: &influxql.RevokeStatement{
				Privilege: influxql.AllPrivileges,
				On:        "testdb",
				User:      "jdoe",
			},
		},

		// REVOKE ALL PRIVILEGES
		{
			s: `REVOKE ALL PRIVILEGES ON testdb FROM jdoe`,
			stmt: &influxql.RevokeStatement{
				Privilege: influxql.AllPrivileges,
				On:        "testdb",
				User:      "jdoe",
			},
		},

		// REVOKE cluster admin
		{
			s: `REVOKE ALL FROM jdoe`,
			stmt: &influxql.RevokeStatement{
				Privilege: influxql.AllPrivileges,
				User:      "jdoe",
			},
		},

		// CREATE RETENTION POLICY
		{
			s: `CREATE RETENTION POLICY policy1 ON testdb DURATION 1h REPLICATION 2`,
			stmt: &influxql.CreateRetentionPolicyStatement{
				Name:        "policy1",
				Database:    "testdb",
				Duration:    time.Hour,
				Replication: 2,
			},
		},

		// CREATE RETENTION POLICY ... DEFAULT
		{
			s: `CREATE RETENTION POLICY policy1 ON testdb DURATION 2m REPLICATION 4 DEFAULT`,
			stmt: &influxql.CreateRetentionPolicyStatement{
				Name:        "policy1",
				Database:    "testdb",
				Duration:    2 * time.Minute,
				Replication: 4,
				Default:     true,
			},
		},

		// ALTER RETENTION POLICY
		{
			s:    `ALTER RETENTION POLICY policy1 ON testdb DURATION 1m REPLICATION 4 DEFAULT`,
			stmt: newAlterRetentionPolicyStatement("policy1", "testdb", time.Minute, 4, true),
		},

		// ALTER RETENTION POLICY with options in reverse order
		{
			s:    `ALTER RETENTION POLICY policy1 ON testdb DEFAULT REPLICATION 4 DURATION 1m`,
			stmt: newAlterRetentionPolicyStatement("policy1", "testdb", time.Minute, 4, true),
		},

		// ALTER RETENTION POLICY without optional DURATION
		{
			s:    `ALTER RETENTION POLICY policy1 ON testdb DEFAULT REPLICATION 4`,
			stmt: newAlterRetentionPolicyStatement("policy1", "testdb", -1, 4, true),
		},

		// ALTER RETENTION POLICY without optional REPLICATION
		{
			s:    `ALTER RETENTION POLICY policy1 ON testdb DEFAULT`,
			stmt: newAlterRetentionPolicyStatement("policy1", "testdb", -1, -1, true),
		},

		// ALTER RETENTION POLICY without optional DEFAULT
		{
			s:    `ALTER RETENTION POLICY policy1 ON testdb REPLICATION 4`,
			stmt: newAlterRetentionPolicyStatement("policy1", "testdb", -1, 4, false),
		},

		// Errors
		{s: ``, err: `found EOF, expected SELECT at line 1, char 1`},
		{s: `SELECT`, err: `found EOF, expected identifier, string, number, bool at line 1, char 8`},
		{s: `blah blah`, err: `found blah, expected SELECT at line 1, char 1`},
		{s: `SELECT field1 X`, err: `found X, expected FROM at line 1, char 15`},
		{s: `SELECT field1 FROM "series" WHERE X +;`, err: `found ;, expected identifier, string, number, bool at line 1, char 38`},
		{s: `SELECT field1 FROM myseries GROUP`, err: `found EOF, expected BY at line 1, char 35`},
		{s: `SELECT field1 FROM myseries LIMIT`, err: `found EOF, expected number at line 1, char 35`},
		{s: `SELECT field1 FROM myseries LIMIT 10.5`, err: `fractional parts not allowed in LIMIT at line 1, char 35`},
		{s: `SELECT field1 FROM myseries LIMIT 0`, err: `LIMIT must be > 0 at line 1, char 35`},
		{s: `SELECT field1 FROM myseries OFFSET`, err: `found EOF, expected number at line 1, char 36`},
		{s: `SELECT field1 FROM myseries OFFSET 10.5`, err: `fractional parts not allowed in OFFSET at line 1, char 36`},
		{s: `SELECT field1 FROM myseries OFFSET 0`, err: `OFFSET must be > 0 at line 1, char 36`},
		{s: `SELECT field1 FROM myseries ORDER`, err: `found EOF, expected BY at line 1, char 35`},
		{s: `SELECT field1 FROM myseries ORDER BY /`, err: `found /, expected identifier, ASC, or DESC at line 1, char 38`},
		{s: `SELECT field1 FROM myseries ORDER BY 1`, err: `found 1, expected identifier, ASC, or DESC at line 1, char 38`},
		{s: `SELECT field1 AS`, err: `found EOF, expected identifier at line 1, char 18`},
		{s: `SELECT field1 FROM 12`, err: `found 12, expected identifier at line 1, char 20`},
		{s: `SELECT field1 FROM myseries GROUP BY *`, err: `found *, expected identifier, string, number, bool at line 1, char 38`},
		{s: `SELECT 1000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000 FROM myseries`, err: `unable to parse number at line 1, char 8`},
		{s: `SELECT 10.5h FROM myseries`, err: `found h, expected FROM at line 1, char 12`},
		{s: `DELETE`, err: `found EOF, expected FROM at line 1, char 8`},
		{s: `DELETE FROM`, err: `found EOF, expected identifier at line 1, char 13`},
		{s: `DELETE FROM myseries WHERE`, err: `found EOF, expected identifier, string, number, bool at line 1, char 28`},
		{s: `DROP SERIES`, err: `found EOF, expected identifier at line 1, char 13`},
		{s: `SHOW CONTINUOUS`, err: `found EOF, expected QUERIES at line 1, char 17`},
		{s: `SHOW RETENTION`, err: `found EOF, expected POLICIES at line 1, char 16`},
		{s: `SHOW RETENTION POLICIES`, err: `found EOF, expected identifier at line 1, char 25`},
		{s: `SHOW FOO`, err: `found FOO, expected CONTINUOUS, DATABASES, FIELD, MEASUREMENTS, RETENTION, SERIES, TAG, USERS at line 1, char 6`},
		{s: `DROP CONTINUOUS`, err: `found EOF, expected QUERY at line 1, char 17`},
		{s: `DROP CONTINUOUS QUERY`, err: `found EOF, expected identifier at line 1, char 23`},
		{s: `DROP FOO`, err: `found FOO, expected SERIES, CONTINUOUS at line 1, char 6`},
		{s: `DROP DATABASE`, err: `found EOF, expected identifier at line 1, char 15`},
		{s: `DROP RETENTION`, err: `found EOF, expected POLICY at line 1, char 16`},
		{s: `DROP RETENTION POLICY`, err: `found EOF, expected identifier at line 1, char 23`},
		{s: `DROP RETENTION POLICY "1h.cpu"`, err: `found EOF, expected ON at line 1, char 32`},
		{s: `DROP RETENTION POLICY "1h.cpu" ON`, err: `found EOF, expected identifier at line 1, char 35`},
		{s: `DROP USER`, err: `found EOF, expected identifier at line 1, char 11`},
		{s: `CREATE USER testuser`, err: `found EOF, expected WITH at line 1, char 22`},
		{s: `CREATE USER testuser WITH`, err: `found EOF, expected PASSWORD at line 1, char 27`},
		{s: `CREATE USER testuser WITH PASSWORD`, err: `found EOF, expected string at line 1, char 36`},
		{s: `CREATE USER testuser WITH PASSWORD 'pwd' WITH`, err: `found EOF, expected ALL at line 1, char 47`},
		{s: `CREATE USER testuser WITH PASSWORD 'pwd' WITH ALL`, err: `found EOF, expected PRIVILEGES at line 1, char 51`},
		{s: `GRANT`, err: `found EOF, expected READ, WRITE, ALL [PRIVILEGES] at line 1, char 7`},
		{s: `GRANT BOGUS`, err: `found BOGUS, expected READ, WRITE, ALL [PRIVILEGES] at line 1, char 7`},
		{s: `GRANT READ`, err: `found EOF, expected ON at line 1, char 12`},
		{s: `GRANT READ TO jdoe`, err: `found TO, expected ON at line 1, char 12`},
		{s: `GRANT READ ON`, err: `found EOF, expected identifier at line 1, char 15`},
		{s: `GRANT READ ON testdb`, err: `found EOF, expected TO at line 1, char 22`},
		{s: `GRANT READ ON testdb TO`, err: `found EOF, expected identifier at line 1, char 25`}, {s: `GRANT`, err: `found EOF, expected READ, WRITE, ALL [PRIVILEGES] at line 1, char 7`},
		{s: `REVOKE BOGUS`, err: `found BOGUS, expected READ, WRITE, ALL [PRIVILEGES] at line 1, char 8`},
		{s: `REVOKE READ`, err: `found EOF, expected ON at line 1, char 13`},
		{s: `REVOKE READ TO jdoe`, err: `found TO, expected ON at line 1, char 13`},
		{s: `REVOKE READ ON`, err: `found EOF, expected identifier at line 1, char 16`},
		{s: `REVOKE READ ON testdb`, err: `found EOF, expected FROM at line 1, char 23`},
		{s: `REVOKE READ ON testdb FROM`, err: `found EOF, expected identifier at line 1, char 28`},
		{s: `CREATE RETENTION`, err: `found EOF, expected POLICY at line 1, char 18`},
		{s: `CREATE RETENTION POLICY`, err: `found EOF, expected identifier at line 1, char 25`},
		{s: `CREATE RETENTION POLICY policy1`, err: `found EOF, expected ON at line 1, char 33`},
		{s: `CREATE RETENTION POLICY policy1 ON`, err: `found EOF, expected identifier at line 1, char 36`},
		{s: `CREATE RETENTION POLICY policy1 ON testdb`, err: `found EOF, expected DURATION at line 1, char 43`},
		{s: `CREATE RETENTION POLICY policy1 ON testdb DURATION`, err: `found EOF, expected duration at line 1, char 52`},
		{s: `CREATE RETENTION POLICY policy1 ON testdb DURATION bad`, err: `found bad, expected duration at line 1, char 52`},
		{s: `CREATE RETENTION POLICY policy1 ON testdb DURATION 1h`, err: `found EOF, expected REPLICATION at line 1, char 54`},
		{s: `CREATE RETENTION POLICY policy1 ON testdb DURATION 1h REPLICATION`, err: `found EOF, expected number at line 1, char 67`},
		{s: `CREATE RETENTION POLICY policy1 ON testdb DURATION 1h REPLICATION 3.14`, err: `number must be an integer at line 1, char 67`},
		{s: `CREATE RETENTION POLICY policy1 ON testdb DURATION 1h REPLICATION 0`, err: `invalid value 0: must be 1 <= n <= 2147483647 at line 1, char 67`},
		{s: `CREATE RETENTION POLICY policy1 ON testdb DURATION 1h REPLICATION bad`, err: `found bad, expected number at line 1, char 67`},
		{s: `ALTER`, err: `found EOF, expected RETENTION at line 1, char 7`},
		{s: `ALTER RETENTION`, err: `found EOF, expected POLICY at line 1, char 17`},
		{s: `ALTER RETENTION POLICY`, err: `found EOF, expected identifier at line 1, char 24`},
		{s: `ALTER RETENTION POLICY policy1`, err: `found EOF, expected ON at line 1, char 32`},
		{s: `ALTER RETENTION POLICY policy1 ON`, err: `found EOF, expected identifier at line 1, char 35`},
		{s: `ALTER RETENTION POLICY policy1 ON testdb`, err: `found EOF, expected DURATION, RETENTION, DEFAULT at line 1, char 42`},
	}

	for i, tt := range tests {
		stmt, err := influxql.NewParser(strings.NewReader(tt.s)).ParseStatement()

		// if it's a CQ, there is a non-exported field that gets memoized during parsing that needs to be set
		if _, ok := stmt.(*influxql.CreateContinuousQueryStatement); ok {
			tt.stmt.(*influxql.CreateContinuousQueryStatement).Source.GroupByInterval()
		}

		if !reflect.DeepEqual(tt.err, errstring(err)) {
			t.Errorf("%d. %q: error mismatch:\n  exp=%s\n  got=%s\n\n", i, tt.s, tt.err, err)
		} else if tt.err == "" && !reflect.DeepEqual(tt.stmt, stmt) {
			t.Errorf("%d. %q\n\nstmt mismatch:\n\nexp=%#v\n\ngot=%#v\n\n", i, tt.s, tt.stmt, stmt)
		}
	}
}

// Ensure the parser can parse expressions into an AST.
func TestParser_ParseExpr(t *testing.T) {
	var tests = []struct {
		s    string
		expr influxql.Expr
		err  string
	}{
		// Primitives
		{s: `100`, expr: &influxql.NumberLiteral{Val: 100}},
		{s: `'foo bar'`, expr: &influxql.StringLiteral{Val: "foo bar"}},
		{s: `true`, expr: &influxql.BooleanLiteral{Val: true}},
		{s: `false`, expr: &influxql.BooleanLiteral{Val: false}},
		{s: `my_ident`, expr: &influxql.VarRef{Val: "my_ident"}},
		{s: `'2000-01-01 00:00:00'`, expr: &influxql.TimeLiteral{Val: mustParseTime("2000-01-01T00:00:00Z")}},
		{s: `'2000-01-01 00:00:00.232'`, expr: &influxql.TimeLiteral{Val: mustParseTime("2000-01-01T00:00:00.232Z")}},
		{s: `'2000-01-32 00:00:00'`, err: `unable to parse datetime at line 1, char 1`},
		{s: `'2000-01-01'`, expr: &influxql.TimeLiteral{Val: mustParseTime("2000-01-01T00:00:00Z")}},
		{s: `'2000-01-99'`, err: `unable to parse date at line 1, char 1`},

		// Simple binary expression
		{
			s: `1 + 2`,
			expr: &influxql.BinaryExpr{
				Op:  influxql.ADD,
				LHS: &influxql.NumberLiteral{Val: 1},
				RHS: &influxql.NumberLiteral{Val: 2},
			},
		},

		// Binary expression with LHS precedence
		{
			s: `1 * 2 + 3`,
			expr: &influxql.BinaryExpr{
				Op: influxql.ADD,
				LHS: &influxql.BinaryExpr{
					Op:  influxql.MUL,
					LHS: &influxql.NumberLiteral{Val: 1},
					RHS: &influxql.NumberLiteral{Val: 2},
				},
				RHS: &influxql.NumberLiteral{Val: 3},
			},
		},

		// Binary expression with RHS precedence
		{
			s: `1 + 2 * 3`,
			expr: &influxql.BinaryExpr{
				Op:  influxql.ADD,
				LHS: &influxql.NumberLiteral{Val: 1},
				RHS: &influxql.BinaryExpr{
					Op:  influxql.MUL,
					LHS: &influxql.NumberLiteral{Val: 2},
					RHS: &influxql.NumberLiteral{Val: 3},
				},
			},
		},

		// Binary expression with LHS paren group.
		{
			s: `(1 + 2) * 3`,
			expr: &influxql.BinaryExpr{
				Op: influxql.MUL,
				LHS: &influxql.ParenExpr{
					Expr: &influxql.BinaryExpr{
						Op:  influxql.ADD,
						LHS: &influxql.NumberLiteral{Val: 1},
						RHS: &influxql.NumberLiteral{Val: 2},
					},
				},
				RHS: &influxql.NumberLiteral{Val: 3},
			},
		},

		// Binary expression with no precedence, tests left associativity.
		{
			s: `1 * 2 * 3`,
			expr: &influxql.BinaryExpr{
				Op: influxql.MUL,
				LHS: &influxql.BinaryExpr{
					Op:  influxql.MUL,
					LHS: &influxql.NumberLiteral{Val: 1},
					RHS: &influxql.NumberLiteral{Val: 2},
				},
				RHS: &influxql.NumberLiteral{Val: 3},
			},
		},

		// Complex binary expression.
		{
			s: `value + 3 < 30 AND 1 + 2 OR true`,
			expr: &influxql.BinaryExpr{
				Op: influxql.OR,
				LHS: &influxql.BinaryExpr{
					Op: influxql.AND,
					LHS: &influxql.BinaryExpr{
						Op: influxql.LT,
						LHS: &influxql.BinaryExpr{
							Op:  influxql.ADD,
							LHS: &influxql.VarRef{Val: "value"},
							RHS: &influxql.NumberLiteral{Val: 3},
						},
						RHS: &influxql.NumberLiteral{Val: 30},
					},
					RHS: &influxql.BinaryExpr{
						Op:  influxql.ADD,
						LHS: &influxql.NumberLiteral{Val: 1},
						RHS: &influxql.NumberLiteral{Val: 2},
					},
				},
				RHS: &influxql.BooleanLiteral{Val: true},
			},
		},

		// Function call (empty)
		{
			s: `my_func()`,
			expr: &influxql.Call{
				Name: "my_func",
			},
		},

		// Function call (multi-arg)
		{
			s: `my_func(1, 2 + 3)`,
			expr: &influxql.Call{
				Name: "my_func",
				Args: []influxql.Expr{
					&influxql.NumberLiteral{Val: 1},
					&influxql.BinaryExpr{
						Op:  influxql.ADD,
						LHS: &influxql.NumberLiteral{Val: 2},
						RHS: &influxql.NumberLiteral{Val: 3},
					},
				},
			},
		},
	}

	for i, tt := range tests {
		expr, err := influxql.NewParser(strings.NewReader(tt.s)).ParseExpr()
		if !reflect.DeepEqual(tt.err, errstring(err)) {
			t.Errorf("%d. %q: error mismatch:\n  exp=%s\n  got=%s\n\n", i, tt.s, tt.err, err)
		} else if tt.err == "" && !reflect.DeepEqual(tt.expr, expr) {
			t.Errorf("%d. %q\n\nexpr mismatch:\n\nexp=%#v\n\ngot=%#v\n\n", i, tt.s, tt.expr, expr)
		}
	}
}

// Ensure a time duration can be parsed.
func TestParseDuration(t *testing.T) {
	var tests = []struct {
		s   string
		d   time.Duration
		err string
	}{
		{s: `3`, d: 3 * time.Microsecond},
		{s: `1000`, d: 1000 * time.Microsecond},
		{s: `10u`, d: 10 * time.Microsecond},
		{s: `10µ`, d: 10 * time.Microsecond},
		{s: `15ms`, d: 15 * time.Millisecond},
		{s: `100s`, d: 100 * time.Second},
		{s: `2m`, d: 2 * time.Minute},
		{s: `2h`, d: 2 * time.Hour},
		{s: `2d`, d: 2 * 24 * time.Hour},
		{s: `2w`, d: 2 * 7 * 24 * time.Hour},

		{s: ``, err: "invalid duration"},
		{s: `w`, err: "invalid duration"},
		{s: `1.2w`, err: "invalid duration"},
		{s: `10x`, err: "invalid duration"},
	}

	for i, tt := range tests {
		d, err := influxql.ParseDuration(tt.s)
		if !reflect.DeepEqual(tt.err, errstring(err)) {
			t.Errorf("%d. %q: error mismatch:\n  exp=%s\n  got=%s\n\n", i, tt.s, tt.err, err)
		} else if tt.d != d {
			t.Errorf("%d. %q\n\nduration mismatch:\n\nexp=%#v\n\ngot=%#v\n\n", i, tt.s, tt.d, d)
		}
	}
}

// Ensure a time duration can be formatted.
func TestFormatDuration(t *testing.T) {
	var tests = []struct {
		d time.Duration
		s string
	}{
		{d: 3 * time.Microsecond, s: `3`},
		{d: 1001 * time.Microsecond, s: `1001`},
		{d: 15 * time.Millisecond, s: `15ms`},
		{d: 100 * time.Second, s: `100s`},
		{d: 2 * time.Minute, s: `2m`},
		{d: 2 * time.Hour, s: `2h`},
		{d: 2 * 24 * time.Hour, s: `2d`},
		{d: 2 * 7 * 24 * time.Hour, s: `2w`},
	}

	for i, tt := range tests {
		s := influxql.FormatDuration(tt.d)
		if tt.s != s {
			t.Errorf("%d. %v: mismatch: %s != %s", i, tt.d, tt.s, s)
		}
	}
}

// Ensure a string can be quoted.
func TestQuote(t *testing.T) {
	for i, tt := range []struct {
		in  string
		out string
	}{
		{``, `''`},
		{`foo`, `'foo'`},
		{"foo\nbar", `'foo\nbar'`},
		{`foo bar\\`, `'foo bar\\\\'`},
		{`'foo'`, `'\'foo\''`},
	} {
		if out := influxql.QuoteString(tt.in); tt.out != out {
			t.Errorf("%d. %s: mismatch: %s != %s", i, tt.in, tt.out, out)
		}
	}
}

// Ensure an identifier's segments can be quoted.
func TestQuoteIdent(t *testing.T) {
	for i, tt := range []struct {
		ident []string
		s     string
	}{
		{[]string{``}, `""`},
		{[]string{`foo`, `bar`}, `"foo"."bar"`},
		{[]string{`foo bar`, `baz`}, `"foo bar"."baz"`},
		{[]string{`foo.bar`, `baz`}, `"foo.bar"."baz"`},
	} {
		if s := influxql.QuoteIdent(tt.ident); tt.s != s {
			t.Errorf("%d. %s: mismatch: %s != %s", i, tt.ident, tt.s, s)
		}
	}
}

func BenchmarkParserParseStatement(b *testing.B) {
	b.ReportAllocs()
	s := `SELECT field FROM "series" WHERE value > 10`
	for i := 0; i < b.N; i++ {
		if stmt, err := influxql.NewParser(strings.NewReader(s)).ParseStatement(); err != nil {
			b.Fatalf("unexpected error: %s", err)
		} else if stmt == nil {
			b.Fatalf("expected statement: %s", stmt)
		}
	}
	b.SetBytes(int64(len(s)))
}

// MustParseSelectStatement parses a select statement. Panic on error.
func MustParseSelectStatement(s string) *influxql.SelectStatement {
	stmt, err := influxql.NewParser(strings.NewReader(s)).ParseStatement()
	if err != nil {
		panic(err.Error())
	}
	return stmt.(*influxql.SelectStatement)
}

// MustParseExpr parses an expression. Panic on error.
func MustParseExpr(s string) influxql.Expr {
	expr, err := influxql.NewParser(strings.NewReader(s)).ParseExpr()
	if err != nil {
		panic(err.Error())
	}
	return expr
}

// errstring converts an error to its string representation.
func errstring(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}

// newAlterRetentionPolicyStatement creates an initialized AlterRetentionPolicyStatement.
func newAlterRetentionPolicyStatement(name string, DB string, d time.Duration, replication int, dfault bool) *influxql.AlterRetentionPolicyStatement {
	stmt := &influxql.AlterRetentionPolicyStatement{
		Name:     name,
		Database: DB,
		Default:  dfault,
	}

	if d > -1 {
		stmt.Duration = &d
	}

	if replication > -1 {
		stmt.Replication = &replication
	}

	return stmt
}
