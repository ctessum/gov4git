package account

import (
	"context"

	"github.com/gov4git/gov4git/proto"
	"github.com/gov4git/gov4git/proto/gov"
	"github.com/gov4git/gov4git/proto/history"
	"github.com/gov4git/gov4git/proto/kv"
	"github.com/gov4git/lib4git/git"
	"github.com/gov4git/lib4git/must"
	"github.com/gov4git/lib4git/ns"
)

type AccountID string

func AccountIDFromNS(p ns.NS) AccountID {
	return AccountID(p.GitPath())
}

func AccountIDFromLine(line Line) AccountID {
	return AccountID(line)
}

func (x AccountID) String() string {
	return string(x)
}

type Account struct {
	ID     AccountID     `json:"id"`
	Owner  OwnerID       `json:"owner"`
	Assets AssetHoldings `json:"assets"`
}

func (a *Account) Deposit(ctx context.Context, h Holding) {
	a.Assets.Deposit(ctx, h)
}

func (a *Account) Withdraw(ctx context.Context, h Holding) {
	a.Assets.Withdraw(ctx, h)
}

func (a *Account) Balance(asset Asset) Holding {
	return a.Assets.Balance(asset)
}

type AssetHoldings map[Asset]Holding

func (x AssetHoldings) Balance(asset Asset) Holding {
	if h, ok := x[asset]; ok {
		return h
	} else {
		return ZeroHolding(asset)
	}
}

func (x AssetHoldings) Deposit(ctx context.Context, h Holding) {
	if g, ok := x[h.Asset]; ok {
		d := SumHolding(ctx, g, h)
		must.Assertf(ctx, d.Quantity >= 0, "insufficient funds")
		x[h.Asset] = d
	} else {
		d := h
		must.Assertf(ctx, d.Quantity >= 0, "no funds")
		x[h.Asset] = d
	}
}

func (x AssetHoldings) Withdraw(ctx context.Context, h Holding) {
	x.Deposit(ctx, NegHolding(h))
}

func NewAccount(id AccountID, owner OwnerID) *Account {
	return &Account{
		ID:     id,
		Owner:  owner,
		Assets: AssetHoldings{},
	}
}

var (
	accountKV = kv.KV[AccountID, *Account]{}
	accountNS = proto.RootNS.Append("account")
)

func Create(
	ctx context.Context,
	addr gov.Address,
	id AccountID,
	owner OwnerID,
	note string,
) {
	cloned := gov.Clone(ctx, addr)
	Create_StageOnly(ctx, cloned, id, owner, note)
	proto.Commitf(ctx, cloned, "account_create", "create account %v (%v)", id, note)
	cloned.Push(ctx)
}

func Create_StageOnly(
	ctx context.Context,
	cloned gov.Cloned,
	id AccountID,
	owner OwnerID,
	note string,
) {
	must.Assertf(ctx, !Exists_Local(ctx, cloned, id), "account %v already exists", id)
	set_StageOnly(ctx, cloned, id, NewAccount(id, owner))
	history.Log_StageOnly(ctx, cloned, &history.Event{
		Op: &history.Op{
			Op:     "account_create",
			Note:   note,
			Args:   history.M{"id": id, "owner": owner},
			Result: nil,
		},
	})
}

func Exists_Local(
	ctx context.Context,
	cloned gov.Cloned,
	id AccountID,
) bool {
	err := must.Try(func() { Get_Local(ctx, cloned, id) })
	switch {
	case err == nil:
		return true
	case git.IsNotExist(err):
		return false
	default:
		must.Panic(ctx, err)
		return false
	}
}

func Get(
	ctx context.Context,
	addr gov.Address,
	id AccountID,
) *Account {
	cloned := gov.Clone(ctx, addr)
	return Get_Local(ctx, cloned, id)
}

func Get_Local(
	ctx context.Context,
	cloned gov.Cloned,
	id AccountID,
) *Account {
	return accountKV.Get(ctx, accountNS, cloned.Tree(), id)
}

func Transfer(
	ctx context.Context,
	addr gov.Address,
	from AccountID,
	to AccountID,
	amount Holding,
	note string,
) {
	cloned := gov.Clone(ctx, addr)
	Transfer_StageOnly(ctx, cloned, from, to, amount, note)
	proto.Commitf(ctx, cloned, "account_transfer", "transfer %v from %v to %v (%v)", amount, from, to, note)
	cloned.Push(ctx)
}

func Transfer_StageOnly(
	ctx context.Context,
	cloned gov.Cloned,
	from AccountID,
	to AccountID,
	amount Holding,
	note string,
) {
	Withdraw_StageOnly(history.Mute(ctx), cloned, from, amount, note)
	Deposit_StageOnly(history.Mute(ctx), cloned, to, amount, note)
	history.Log_StageOnly(ctx, cloned, &history.Event{
		Op: &history.Op{
			Op:     "account_transfer",
			Note:   note,
			Args:   history.M{"from": from, "to": to, "amount": amount},
			Result: nil,
		},
	})
}

func TryTransfer_StageOnly(
	ctx context.Context,
	cloned gov.Cloned,
	from AccountID,
	to AccountID,
	amount Holding,
	note string,
) error {
	return must.Try(func() { Transfer_StageOnly(ctx, cloned, from, to, amount, note) })
}

func Deposit(
	ctx context.Context,
	addr gov.Address,
	to AccountID,
	amount Holding,
	note string,
) {
	cloned := gov.Clone(ctx, addr)
	Deposit_StageOnly(ctx, cloned, to, amount, note)
	proto.Commitf(ctx, cloned, "account_deposit", "deposit %v to %v (%v)", amount, to, note)
	cloned.Push(ctx)
}

func Deposit_StageOnly(
	ctx context.Context,
	cloned gov.Cloned,
	to AccountID,
	amount Holding,
	note string,
) {
	a := Get_Local(ctx, cloned, to)
	a.Deposit(ctx, amount)
	set_StageOnly(ctx, cloned, to, a)
	history.Log_StageOnly(ctx, cloned, &history.Event{
		Op: &history.Op{
			Op:     "account_deposit",
			Note:   note,
			Args:   history.M{"to": to, "amount": amount},
			Result: nil,
		},
	})
}

func Withdraw(
	ctx context.Context,
	addr gov.Address,
	from AccountID,
	amount Holding,
	note string,
) {
	cloned := gov.Clone(ctx, addr)
	Withdraw_StageOnly(ctx, cloned, from, amount, note)
	proto.Commitf(ctx, cloned, "account_withdraw", "withdraw %v from %v (%v)", amount, from, note)
	cloned.Push(ctx)
}

func Withdraw_StageOnly(
	ctx context.Context,
	cloned gov.Cloned,
	from AccountID,
	amount Holding,
	note string,
) {
	a := Get_Local(ctx, cloned, from)
	a.Withdraw(ctx, amount)
	set_StageOnly(ctx, cloned, from, a)
	history.Log_StageOnly(ctx, cloned, &history.Event{
		Op: &history.Op{
			Op:     "account_withdraw",
			Note:   note,
			Args:   history.M{"from": from, "amount": amount},
			Result: nil,
		},
	})
}

func set_StageOnly(
	ctx context.Context,
	cloned gov.Cloned,
	id AccountID,
	account *Account,
) {
	accountKV.Set(ctx, accountNS, cloned.Tree(), id, account)
}

func List(
	ctx context.Context,
	addr gov.Address,
) []AccountID {
	cloned := gov.Clone(ctx, addr)
	return List_Local(ctx, cloned)
}

func List_Local(
	ctx context.Context,
	cloned gov.Cloned,
) []AccountID {
	return accountKV.ListKeys(ctx, accountNS, cloned.Tree())
}
