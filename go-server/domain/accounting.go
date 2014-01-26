package domain

import (
	//"log"
	"fmt"
	"reflect"
	"strings"
	"time"
	"appengine"
	"appengine/datastore"
)

type ChartOfAccounts struct {
	Key *datastore.Key `json:"_id",datastore:"-"`
	Name string `json:"name"`
	RetainedEarningsAccount *datastore.Key `json:"retainedEarningsAccount"`
	User *datastore.Key `json:"user"`
	AsOf time.Time `json:"timestamp"`
}

func (coa *ChartOfAccounts) ValidationMessage(_ appengine.Context, _ map[string]string) string {
	if len(strings.TrimSpace(coa.Name)) == 0 {
		return "The name must be informed"
	}
	return ""
}

type Account struct {
	Key *datastore.Key `json:"_id",datastore:"-"`
	Number string `json:"number"`
	Name string `json:"name"`
	Tags []string `json:"tags"`
	Parent *datastore.Key `json:"parent"`
	User *datastore.Key `json:"user"`
	AsOf time.Time `json:"timestamp"`
}

var inheritedProperties = map[string]string{
	"balanceSheet": "financial statement",
	"incomeStatement": "financial statement",
	"operating": "income statement attribute",
	"deduction": "income statement attribute",
	"salesTax": "income statement attribute",
	"cost": "income statement attribute",
	"nonOperatingTax": "income statement attribute",
	"incomeTax": "income statement attribute",
	"dividends": "income statement attribute"}

func (account *Account) ValidationMessage(c appengine.Context, param map[string]string) string {
	if len(strings.TrimSpace(account.Number)) == 0 {
		return "The number must be informed"
	}
	if len(strings.TrimSpace(account.Name)) == 0 {
		return "The name must be informed"
	}
	if !contains(account.Tags, "balanceSheet") && !contains(account.Tags, "incomeStatement") {
		return "The financial statement must be informed"
	}
	if contains(account.Tags, "balanceSheet") && contains(account.Tags, "incomeStatement") {
		return "The statement must be either balance sheet or income statement"
	}
	if !contains(account.Tags, "debitBalance") && !contains(account.Tags, "creditBalance") {
		return "The normal balance must be informed"
	}
	if contains(account.Tags, "debitBalance") && contains(account.Tags, "creditBalance") {
		return "The normal balance must be either debit or credit"
	}
	count := 0
	for _, p := range account.Tags {
		if inheritedProperties[p] == "income statement attribute" {
			count++
		}
	}
	if count > 1 {
		return "Only one income statement attribute is allowed"
	}
	if account.Key == nil {
		coaKey, err := datastore.DecodeKey(param["coa"])
		if err != nil {
			return err.Error()
		}
		q := datastore.NewQuery("Account").Ancestor(coaKey).Filter("Number = ", account.Number).KeysOnly()
		keys, err := q.GetAll(c, nil)
		if err != nil {
			return err.Error()
		}
		if len(keys) != 0 {
			return "An account with this number already exists"
		}		
	}
	if account.Parent != nil {
		var parent Account
		if err := datastore.Get(c, account.Parent, &parent); err != nil {
			return err.Error()
		}
		if !strings.HasPrefix(account.Number, parent.Number) {
			return "The number must start with parent's number"
		}
		for key, value := range inheritedProperties {
			if contains(parent.Tags, key) && !contains(account.Tags, key) {
				return "The " + value + " must be same as the parent"
			}
		}
		if account.Parent.Parent().String() != account.Key.Parent().String() {
			return "The account's parent must belong to the same chart of accounts of the account"
		}
	}
	return ""
}

type Transaction struct {
	Key *datastore.Key `json:"_id",datastore:"-"`
	Debits []Entry `json:"debits"`
	Credits []Entry `json:"credits"`
	Date time.Time `json:"date`
	Memo string `json:"memo"`
	Tags []string `json:"tags"`
	User *datastore.Key `json:"user"`
	AsOf time.Time `json:"timestamp"`
}

type Entry struct {
	Account *datastore.Key `json:"account"`
	Value float64 `json:"value"`
}

func (transaction *Transaction) ValidationMessage(c appengine.Context, param map[string]string) string {
	if len(transaction.Debits) == 0 {
		return "At least one debit must be informed"
	}
	if len(transaction.Credits) == 0 {
		return "At least one credit must be informed"
	}
	if transaction.Date.IsZero() {
		return "The date must be informed"
	}
	if len(strings.TrimSpace(transaction.Memo)) == 0 {
		return "The memo must be informed"
	}
	ev := func(arr []Entry) (string, float64) {
		sum := 0.0
		for _, e := range arr {
			if m := e.ValidationMessage(c, param); len(m) > 0 {
				return m, 0.0
			}
			sum += e.Value
		}
		return "", sum
	}
	var debitsSum, creditsSum float64
	var m string
	if m, debitsSum = ev(transaction.Debits); len(m) > 0 {
		return m
	}
	if m, creditsSum = ev(transaction.Credits); len(m) > 0 {
		return m
	}
	if debitsSum != creditsSum {
		return "The sum of debit values must be equals to the sum of credit values"
	}
	return ""
}

func (entry *Entry) ValidationMessage(c appengine.Context, param map[string]string) string {
	if entry.Account == nil {
		return "The account must be informed for each entry"
	}
	var account = new(Account)
	if err := datastore.Get(c, entry.Account, account); err != nil {
		return err.Error()
	}
	if account == nil {
		return "Account not found"
	}
	if !contains(account.Tags, "analytic") {
		return "The account must be analytic"
	}
	coaKey, err := datastore.DecodeKey(param["coa"])
	if err != nil {
		return err.Error()
	}
	if entry.Account.Parent().String() != coaKey.String() {
		return "The account must belong to the same chart of accounts of the transaction"
	}

	return ""
}

func AllChartsOfAccounts(c appengine.Context, _ map[string]string, _ *datastore.Key) (interface{}, error) {
	return getAll(c, &[]ChartOfAccounts{}, "ChartOfAccounts", "")
}

func SaveChartOfAccounts(c appengine.Context, m map[string]interface{}, param map[string]string, userKey *datastore.Key) (interface{}, error) {
	coa := &ChartOfAccounts{
		Name: m["name"].(string), 
		User: userKey,
		AsOf: time.Now()}
	_, err := save(c, coa, "ChartOfAccounts", "", param)
	return coa, err
}

func AllAccounts(c appengine.Context, param map[string]string, _ *datastore.Key) (interface{}, error) {
	return getAll(c, &[]Account{}, "Account", param["coa"])
}

func SaveAccount(c appengine.Context, m map[string]interface{}, param map[string]string, userKey *datastore.Key) (item interface{}, err error) {

	account := &Account{
		Number: m["number"].(string), 
		Name: m["name"].(string), 
		Tags: []string{},
		User: userKey,
		AsOf: time.Now()}

	if accountKeyAsString, ok := param["account"]; ok {
		account.Key, err = datastore.DecodeKey(accountKeyAsString)
		if err != nil {
			return
		}
	}

	coaKey, err := datastore.DecodeKey(param["coa"])
	if err != nil {
		return
	}

	var parent []Account
	if parentNumber, ok := m["parent"]; ok {
		q := datastore.NewQuery("Account").Ancestor(coaKey).Filter("Number = ", parentNumber)
		keys, err := q.GetAll(c, &parent)
		if err != nil {
			return nil, err
		}
		if len(keys) == 0 {
			return nil, fmt.Errorf("Parent not found")
		}
		account.Parent = keys[0]
		delete(m, "parent")
	}

	var retainedEarningsAccount bool
	for k, _ := range m {
		if k != "name" && k != "number" {
			if k == "retainedEarnings" {
				retainedEarningsAccount = true
			} else {
				account.Tags = append(account.Tags, k)
			}
		}
	}
	if !contains(account.Tags, "analytic") {
		account.Tags = append(account.Tags, "analytic")
	}

	err = datastore.RunInTransaction(c, func(c appengine.Context) (err error) {

		accountKey, err := save(c, account, "Account", param["coa"], param)
		if err != nil {
			return
		}

		if retainedEarningsAccount {
			coa := new(ChartOfAccounts)
			if err = datastore.Get(c, coaKey, coa); err != nil {
				return
			}
			coa.RetainedEarningsAccount = accountKey
			if _, err = datastore.Put(c, coaKey, coa); err != nil {
				return
			}
		}

		if account.Parent != nil {
			i := indexOf(parent[0].Tags, "analytic")
			if i != -1 {
				parent[0].Tags = append(parent[0].Tags[:i], parent[0].Tags[i+1:]...)
			}
			parent[0].Tags = append(parent[0].Tags, "synthetic")
			if _, err = datastore.Put(c, account.Parent, &parent[0]); err != nil {
				return
			}
		}
		return
	}, nil)
	if err != nil {
		return
	}

	item = account
	return
}

func AllTransactions(c appengine.Context, param map[string]string, _ *datastore.Key) (interface{}, error) {
	return getAll(c, &[]Transaction{}, "Transaction", param["coa"])
}

func SaveTransaction(c appengine.Context, m map[string]interface{}, param map[string]string, userKey *datastore.Key) (item interface{}, err error) {

	transaction := &Transaction{
		Memo: m["memo"].(string),
		AsOf: time.Now(),
		User: userKey}
	transaction.Date, err = time.Parse(time.RFC3339, m["date"].(string))
	if err != nil {
		return
	}

	coaKey, err := datastore.DecodeKey(param["coa"])
	if err != nil {
		return
	}

	entries := func(property string) (result []Entry, err error) {
		for _, each := range m[property].([]interface{}) {

			entry := each.(map[string]interface{})

			q := datastore.NewQuery("Account").Ancestor(coaKey).Filter("Number = ", entry["account"]).KeysOnly()
			var keys []*datastore.Key
			if keys, err = q.GetAll(c, nil); err != nil {
				return
			}
			if len(keys) == 0 {
				return nil, fmt.Errorf("Account '%v' not found", entry["account"])
			}

			result = append(result, Entry{Account: keys[0], Value: entry["value"].(float64)})
		}
		return
	}
	if transaction.Debits, err = entries("debits"); err != nil {
		return
	}
	if transaction.Credits, err = entries("credits"); err != nil {
		return
	}

	transactionKey, err := save(c, transaction, "Transaction", param["coa"], param)
	if err != nil {
		return
	}
	transaction.Key = transactionKey
	item = transaction

	return
}

func getAll(c appengine.Context, items interface{}, kind string, ancestor string) (interface{}, error) {
	q := datastore.NewQuery(kind)
	if len(ancestor) > 0 {
		ancestorKey, err := datastore.DecodeKey(ancestor)
		if err != nil {
			return nil, err
		}
		q = q.Ancestor(ancestorKey)
	}
	keys, err := q.GetAll(c, items)
	v := reflect.ValueOf(items).Elem()
	for i := 0; i < v.Len(); i++ {
		v.Index(i).FieldByName("Key").Set(reflect.ValueOf(keys[i]))
	}
	return items, err
}

func save(c appengine.Context, item interface{}, kind string, ancestor string, param map[string]string) (key *datastore.Key, err error) {
	vm := item.(ValidationMessager).ValidationMessage(c, param)
	if len(vm) > 0 {
		return nil, fmt.Errorf(vm)
	}

	var ancestorKey *datastore.Key
	if len(ancestor) > 0 {
		ancestorKey, err = datastore.DecodeKey(ancestor)
		if err != nil {
			return
		}
	}

	v := reflect.ValueOf(item).Elem()

	if !v.FieldByName("Key").IsNil() {
		key = v.FieldByName("Key").Interface().(*datastore.Key)
	} else {
		key = datastore.NewIncompleteKey(c, kind, ancestorKey)
	}

	key, err = datastore.Put(c, key, item)
	if err != nil {
		return
	}

	if key != nil {
		v.FieldByName("Key").Set(reflect.ValueOf(key))
	}

	return
}

func contains(s []string, e string) bool {
    return indexOf(s, e) != -1
}

func indexOf(s []string, e string) int {
    for i, a := range s { if a == e { return i } }
    return -1
}

type ValidationMessager interface {
	ValidationMessage(appengine.Context, map[string]string) string
}
