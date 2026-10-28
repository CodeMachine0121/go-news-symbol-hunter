package domains

import (
	"cmp"
	"slices"
	"strings"

	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/dto"
	"github.com/CodeMachine0121/go-news-symbol-hunter/internal/domain/models/vo"
)

const maximumSymbolCandidates = 10

// "Apple" should equal "Apple Inc.", so US company names are compared without these trailing words
var usStockCompanySuffixes = map[string]bool{
	"inc": true, "incorporated": true, "corp": true, "corporation": true, "co": true, "company": true,
	"ltd": true, "limited": true, "plc": true, "lp": true, "llc": true, "sa": true, "nv": true, "ag": true, "holdings": true,
}

type SymbolDirectoryDomain struct {
	category vo.MarketCategoryVo
	listings []vo.SymbolListingVo
	nameKeys []string
}

func NewSymbolDirectoryDomain(category vo.MarketCategoryVo, rawListings []vo.SymbolListingVo) SymbolDirectoryDomain {
	listings := []vo.SymbolListingVo{}
	for _, rawListing := range rawListings {
		listing := vo.SymbolListingVo{Symbol: strings.ToUpper(strings.TrimSpace(rawListing.Symbol)), Name: strings.TrimSpace(rawListing.Name), Priority: rawListing.Priority}
		// a Taiwan stock or coin is searched by its name, while a US stock is searched by its symbol and may go unnamed
		if listing.Symbol != "" && (listing.Name != "" || category.Value == vo.MarketCategoryUsStock) {
			listings = append(listings, listing)
		}
	}
	slices.SortStableFunc(listings, func(first vo.SymbolListingVo, second vo.SymbolListingVo) int {
		return cmp.Or(cmp.Compare(first.Priority, second.Priority), strings.Compare(first.Symbol, second.Symbol))
	})
	// an analysis only knows its symbol, which stands for the best-ranked listing, so lower-ranked listings sharing it cannot be analysed
	symbolDirectoryDomain := SymbolDirectoryDomain{category: category}
	listedSymbols := map[string]bool{}
	for _, listing := range listings {
		if !listedSymbols[listing.Symbol] {
			symbolDirectoryDomain.listings = append(symbolDirectoryDomain.listings, listing)
			listedSymbols[listing.Symbol] = true
		}
	}
	for _, listing := range symbolDirectoryDomain.listings {
		symbolDirectoryDomain.nameKeys = append(symbolDirectoryDomain.nameKeys, symbolDirectoryDomain.nameKey(listing.Name))
	}
	return symbolDirectoryDomain
}

func (symbolDirectoryDomain SymbolDirectoryDomain) FindBySymbol(symbol vo.SymbolVo) (vo.ResolvedSymbolVo, bool) {
	for _, listing := range symbolDirectoryDomain.listings {
		if listing.Symbol == symbol.Value {
			return symbolDirectoryDomain.toResolvedSymbol(listing), true
		}
	}
	return vo.ResolvedSymbolVo{}, false
}

// the typed text may be a symbol or a name: an exact symbol or name wins, then a name containing it when only one does;
// when several names contain it, the candidates come back instead
func (symbolDirectoryDomain SymbolDirectoryDomain) Identify(symbol vo.SymbolVo) (vo.ResolvedSymbolVo, []dto.SymbolCandidateDto, bool) {
	typedNameKey := symbolDirectoryDomain.nameKey(symbol.Value)
	for index, listing := range symbolDirectoryDomain.listings {
		isExactName := typedNameKey != "" && symbolDirectoryDomain.nameKeys[index] == typedNameKey
		if listing.Symbol == symbol.Value || isExactName {
			return symbolDirectoryDomain.toResolvedSymbol(listing), nil, true
		}
	}
	partialMatches := []vo.SymbolListingVo{}
	for index, listing := range symbolDirectoryDomain.listings {
		if typedNameKey != "" && strings.Contains(symbolDirectoryDomain.nameKeys[index], typedNameKey) {
			partialMatches = append(partialMatches, listing)
		}
	}
	if len(partialMatches) == 1 {
		return symbolDirectoryDomain.toResolvedSymbol(partialMatches[0]), nil, true
	}
	candidates := []dto.SymbolCandidateDto{}
	for _, listing := range partialMatches[:min(len(partialMatches), maximumSymbolCandidates)] {
		candidates = append(candidates, dto.SymbolCandidateDto{Symbol: listing.Symbol, SymbolName: listing.Name})
	}
	return vo.ResolvedSymbolVo{}, candidates, false
}

func (symbolDirectoryDomain SymbolDirectoryDomain) toResolvedSymbol(listing vo.SymbolListingVo) vo.ResolvedSymbolVo {
	symbol := vo.SymbolVo{Value: listing.Symbol, Category: symbolDirectoryDomain.category}
	switch symbolDirectoryDomain.category.Value {
	case vo.MarketCategoryTwStock:
		return vo.NewTwStockResolvedSymbolVo(symbol, listing.Name)
	case vo.MarketCategoryCrypto:
		return vo.NewCryptoResolvedSymbolVo(symbol, listing.Name)
	default:
		return vo.NewUsStockResolvedSymbolVo(symbol, listing.Name)
	}
}

func (symbolDirectoryDomain SymbolDirectoryDomain) nameKey(name string) string {
	loweredName := strings.ToLower(name)
	if symbolDirectoryDomain.category.Value != vo.MarketCategoryUsStock {
		return strings.Join(strings.Fields(loweredName), " ")
	}
	words := strings.Fields(strings.NewReplacer(",", " ", ".", " ").Replace(loweredName))
	for len(words) > 1 && usStockCompanySuffixes[words[len(words)-1]] {
		words = words[:len(words)-1]
	}
	return strings.Join(words, " ")
}
