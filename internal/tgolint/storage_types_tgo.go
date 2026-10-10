package tgolint

import (
	"go/token"
	"go/types"
	"strconv"
)

func encodeStorageEffectType(
	typ types.Type,
	parameters map[*types.TypeParam]zeroParameter,
) StorageEffectType {
	return encodeStorageEffectTypeSeen(typ, parameters, make(map[types.Type]bool))
}

func encodeStorageEffectTypeSeen(
	typ types.Type,
	parameters map[*types.TypeParam]zeroParameter,
	seen map[types.Type]bool,
) StorageEffectType {
	typ = types.Unalias(typ)
	classified := goTypeOf(typ)
	if classified.Tag() == goTypeTagTypeParameter {
		parameter := classified.TypeParameterPayload().Value
		binding, found := parameters[parameter]
		if !found {
			return StorageEffectType{Kind: storageTypeInterface}
		}
		kind := storageTypeParameter
		if binding.receiver {
			kind = storageTypeReceiver
		}
		return StorageEffectType{Kind: kind, Parameter: binding.index}
	}
	if seen[typ] {
		return StorageEffectType{Kind: storageTypeInterface}
	}
	seen[typ] = true
	defer delete(seen, typ)
	switch classified.Tag() {
	case goTypeTagBasic:
		value := classified.BasicPayload().Value
		return StorageEffectType{Kind: storageTypeBasic, Basic: int(value.Kind())}
	case goTypeTagSlice:
		return storageUnaryType(storageTypeSlice, classified.SlicePayload().Value.Elem(), parameters, seen)
	case goTypeTagArray:
		value := classified.ArrayPayload().Value
		result := storageUnaryType(storageTypeArray, value.Elem(), parameters, seen)
		result.Length = value.Len()
		return result
	case goTypeTagMap:
		value := classified.MapPayload().Value
		return StorageEffectType{Kind: storageTypeMap,
			Key:     []StorageEffectType{encodeStorageEffectTypeSeen(value.Key(), parameters, seen)},
			Element: []StorageEffectType{encodeStorageEffectTypeSeen(value.Elem(), parameters, seen)}}
	case goTypeTagPointer:
		return storageUnaryType(storageTypePointer, classified.PointerPayload().Value.Elem(), parameters, seen)
	case goTypeTagChannel:
		value := classified.ChannelPayload().Value
		result := storageUnaryType(storageTypeChannel, value.Elem(), parameters, seen)
		result.Direction = int(value.Dir())
		return result
	case goTypeTagStruct:
		value := classified.StructPayload().Value
		result := StorageEffectType{Kind: storageTypeStruct}
		for index := 0; index < value.NumFields(); index++ {
			field := value.Field(index)
			result.Fields = append(result.Fields, encodeStorageEffectTypeSeen(field.Type(), parameters, seen))
			result.FieldNames = append(result.FieldNames, field.Name())
			packagePath := ""
			if field.Pkg() != nil {
				packagePath = field.Pkg().Path()
			}
			result.FieldPkgs = append(result.FieldPkgs, packagePath)
			result.FieldTags = append(result.FieldTags, value.Tag(index))
			result.FieldEmbed = append(result.FieldEmbed, field.Embedded())
		}
		return result
	case goTypeTagSignature:
		value := classified.SignaturePayload().Value
		return StorageEffectType{Kind: storageTypeSignature, Variadic: value.Variadic(),
			Parameters: encodeStorageTuple(value.Params(), parameters, seen),
			Results:    encodeStorageTuple(value.Results(), parameters, seen)}
	case goTypeTagNamed:
		value := classified.NamedPayload().Value
		result := StorageEffectType{
			Kind: storageTypeNamed, Name: value.Obj().Name(), Position: int(value.Obj().Pos()),
		}
		if value.Obj().Pkg() != nil {
			result.Package = value.Obj().Pkg().Path()
			result.PackageLevel = value.Obj().Parent() == value.Obj().Pkg().Scope()
		}
		for index := 0; index < value.TypeArgs().Len(); index++ {
			result.Arguments = append(result.Arguments,
				encodeStorageEffectTypeSeen(value.TypeArgs().At(index), parameters, seen))
		}
		result.Underlying = []StorageEffectType{
			encodeStorageEffectTypeSeen(value.Underlying(), parameters, seen),
		}
		return result
	case goTypeTagInterface:
		value := classified.InterfacePayload().Value
		value = value.Complete()
		result := StorageEffectType{Kind: storageTypeInterface}
		for index := 0; index < value.NumExplicitMethods(); index++ {
			method := value.ExplicitMethod(index)
			result.Fields = append(result.Fields,
				encodeStorageEffectTypeSeen(method.Type(), parameters, seen))
			result.FieldNames = append(result.FieldNames, method.Name())
			packagePath := ""
			if method.Pkg() != nil {
				packagePath = method.Pkg().Path()
			}
			result.FieldPkgs = append(result.FieldPkgs, packagePath)
		}
		for index := 0; index < value.NumEmbeddeds(); index++ {
			result.Element = append(result.Element,
				encodeStorageEffectTypeSeen(value.EmbeddedType(index), parameters, seen))
		}
		return result
	default:
		return StorageEffectType{Kind: storageTypeInterface}
	}
}

func storageUnaryType(
	kind int,
	element types.Type,
	parameters map[*types.TypeParam]zeroParameter,
	seen map[types.Type]bool,
) StorageEffectType {
	return StorageEffectType{Kind: kind,
		Element: []StorageEffectType{encodeStorageEffectTypeSeen(element, parameters, seen)}}
}

func encodeStorageTuple(
	tuple *types.Tuple,
	parameters map[*types.TypeParam]zeroParameter,
	seen map[types.Type]bool,
) []StorageEffectType {
	result := make([]StorageEffectType, tuple.Len())
	for index := range result {
		result[index] = encodeStorageEffectTypeSeen(tuple.At(index).Type(), parameters, seen)
	}
	return result
}

func decodeStorageEffectType(
	projection StorageEffectType,
	receiverArguments []types.Type,
	typeArguments []types.Type,
) types.Type {
	return decodeStorageEffectTypeWithNames(
		projection, receiverArguments, typeArguments, nil,
	)
}

func decodeStorageEffectTypeWithNames(
	projection StorageEffectType,
	receiverArguments []types.Type,
	typeArguments []types.Type,
	namedTypes map[string]*types.Named,
) types.Type {
	switch projection.Kind {
	case storageTypeParameter:
		return storageBoundType(typeArguments, projection.Parameter)
	case storageTypeReceiver:
		return storageBoundType(receiverArguments, projection.Parameter)
	case storageTypeBasic:
		return types.Typ[types.BasicKind(projection.Basic)]
	case storageTypeSlice:
		return types.NewSlice(decodeStorageElement(projection, receiverArguments, typeArguments, namedTypes))
	case storageTypeArray:
		return types.NewArray(decodeStorageElement(projection, receiverArguments, typeArguments, namedTypes), projection.Length)
	case storageTypeMap:
		return types.NewMap(
			decodeStorageTypeList(projection.Key, receiverArguments, typeArguments, namedTypes)[0],
			decodeStorageElement(projection, receiverArguments, typeArguments, namedTypes),
		)
	case storageTypePointer:
		return types.NewPointer(decodeStorageElement(projection, receiverArguments, typeArguments, namedTypes))
	case storageTypeChannel:
		return types.NewChan(types.ChanDir(projection.Direction),
			decodeStorageElement(projection, receiverArguments, typeArguments, namedTypes))
	case storageTypeStruct:
		fields := make([]*types.Var, len(projection.Fields))
		for index, field := range decodeStorageTypeList(projection.Fields, receiverArguments, typeArguments, namedTypes) {
			var pkg *types.Package
			if projection.FieldPkgs[index] != "" {
				pkg = types.NewPackage(projection.FieldPkgs[index], projection.FieldPkgs[index])
			}
			fields[index] = types.NewField(0, pkg, projection.FieldNames[index], field, projection.FieldEmbed[index])
		}
		return types.NewStruct(fields, projection.FieldTags)
	case storageTypeSignature:
		parameters := types.NewTuple(storageTypeVars(
			projection.Parameters, receiverArguments, typeArguments, namedTypes)...)
		results := types.NewTuple(storageTypeVars(
			projection.Results, receiverArguments, typeArguments, namedTypes)...)
		return types.NewSignatureType(nil, nil, nil, parameters, results, projection.Variadic)
	case storageTypeNamed:
		return decodeStorageNamed(projection, receiverArguments, typeArguments, namedTypes)
	case storageTypeInterface:
		methods := make([]*types.Func, len(projection.Fields))
		for index, signature := range decodeStorageTypeList(
			projection.Fields, receiverArguments, typeArguments, namedTypes,
		) {
			var pkg *types.Package
			if projection.FieldPkgs[index] != "" {
				pkg = types.NewPackage(projection.FieldPkgs[index], projection.FieldPkgs[index])
			}
			methods[index] = types.NewFunc(0, pkg, projection.FieldNames[index], signature.(*types.Signature))
		}
		embedded := decodeStorageTypeList(projection.Element, receiverArguments, typeArguments, namedTypes)
		return types.NewInterfaceType(methods, embedded).Complete()
	}
	return types.NewInterfaceType(nil, nil).Complete()
}

func storageBoundType(arguments []types.Type, index int) types.Type {
	if index >= 0 && index < len(arguments) && arguments[index] != nil {
		return arguments[index]
	}
	return types.NewInterfaceType(nil, nil).Complete()
}

func decodeStorageElement(
	projection StorageEffectType,
	receiverArguments []types.Type,
	typeArguments []types.Type,
	namedTypes map[string]*types.Named,
) types.Type {
	return decodeStorageTypeList(projection.Element, receiverArguments, typeArguments, namedTypes)[0]
}

func decodeStorageTypeList(
	projections []StorageEffectType,
	receiverArguments []types.Type,
	typeArguments []types.Type,
	namedTypes map[string]*types.Named,
) []types.Type {
	result := make([]types.Type, len(projections))
	for index, projection := range projections {
		result[index] = decodeStorageEffectTypeWithNames(
			projection, receiverArguments, typeArguments, namedTypes,
		)
	}
	return result
}

func storageTypeVars(
	projections []StorageEffectType,
	receiverArguments []types.Type,
	typeArguments []types.Type,
	namedTypes map[string]*types.Named,
) []*types.Var {
	typesList := decodeStorageTypeList(projections, receiverArguments, typeArguments, namedTypes)
	result := make([]*types.Var, len(typesList))
	for index, typ := range typesList {
		result[index] = types.NewVar(0, nil, "", typ)
	}
	return result
}

func decodeStorageNamed(
	projection StorageEffectType,
	receiverArguments []types.Type,
	typeArguments []types.Type,
	namedTypes map[string]*types.Named,
) types.Type {
	key := storageNamedTypeKey(projection.Package, projection.Name)
	var named *types.Named
	if projection.PackageLevel {
		named = namedTypes[key]
	}
	localKey := key + "\x00" + strconv.Itoa(projection.Position)
	if named == nil {
		named = namedTypes[localKey]
	}
	if named == nil {
		var pkg *types.Package
		if projection.Package != "" {
			pkg = types.NewPackage(projection.Package, projection.Package)
		}
		underlying := types.Type(types.NewStruct(nil, nil))
		if len(projection.Underlying) != 0 {
			underlying = decodeStorageEffectTypeWithNames(
				projection.Underlying[0], receiverArguments, typeArguments, namedTypes,
			)
		}
		named = types.NewNamed(
			types.NewTypeName(token.Pos(projection.Position), pkg, projection.Name, nil),
			underlying, nil,
		)
		if namedTypes != nil {
			namedTypes[localKey] = named
		}
	}
	arguments := decodeStorageTypeList(
		projection.Arguments, receiverArguments, typeArguments, namedTypes,
	)
	if len(arguments) == 0 {
		return named
	}
	if named.TypeParams().Len() == 0 {
		parameters := make([]*types.TypeParam, len(arguments))
		for index := range parameters {
			parameters[index] = types.NewTypeParam(
				types.NewTypeName(0, nil, "T", nil),
				types.NewInterfaceType(nil, nil).Complete(),
			)
		}
		named.SetTypeParams(parameters)
	}
	result, err := types.Instantiate(nil, named, arguments, false)
	if err != nil {
		return named
	}
	return result
}

func storageEffectTypeParameters(
	projection StorageEffectType,
	zero bool,
) []StorageEffectType {
	if projection.Kind == storageTypeParameter || projection.Kind == storageTypeReceiver {
		return []StorageEffectType{projection}
	}
	if !zero {
		return nil
	}
	var children []StorageEffectType
	switch projection.Kind {
	case storageTypeArray:
		if projection.Length > 0 {
			children = projection.Element
		}
	case storageTypeStruct:
		children = projection.Fields
	case storageTypeNamed:
		children = projection.Underlying
	}
	var result []StorageEffectType
	seen := make(map[[2]int]bool)
	for _, child := range children {
		for _, parameter := range storageEffectTypeParameters(child, true) {
			key := [2]int{parameter.Kind, parameter.Parameter}
			if !seen[key] {
				seen[key] = true
				result = append(result, parameter)
			}
		}
	}
	return result
}

func storageNamedTypeKey(packagePath, name string) string {
	return packagePath + "\x00" + name
}

func collectStorageNamedTypes(
	pkg *types.Package,
	result map[string]*types.Named,
	seen map[*types.Package]bool,
) {
	if pkg == nil || seen[pkg] {
		return
	}
	seen[pkg] = true
	for _, name := range pkg.Scope().Names() {
		object, ok := pkg.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := types.Unalias(object.Type()).(*types.Named)
		if ok {
			result[storageNamedTypeKey(pkg.Path(), name)] = named.Origin()
		}
	}
	for _, imported := range pkg.Imports() {
		collectStorageNamedTypes(imported, result, seen)
	}
}
