package tgolint

import "go/types"

func encodeStorageEffectType(
	typ types.Type,
	parameters map[*types.TypeParam]zeroParameter,
) StorageEffectType {
	typ = types.Unalias(typ)
	if parameter, ok := typ.(*types.TypeParam); ok {
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
	switch value := typ.(type) {
	case *types.Basic:
		return StorageEffectType{Kind: storageTypeBasic, Basic: int(value.Kind())}
	case *types.Slice:
		return storageUnaryType(storageTypeSlice, value.Elem(), parameters)
	case *types.Array:
		result := storageUnaryType(storageTypeArray, value.Elem(), parameters)
		result.Length = value.Len()
		return result
	case *types.Map:
		return StorageEffectType{Kind: storageTypeMap,
			Key:     []StorageEffectType{encodeStorageEffectType(value.Key(), parameters)},
			Element: []StorageEffectType{encodeStorageEffectType(value.Elem(), parameters)}}
	case *types.Pointer:
		return storageUnaryType(storageTypePointer, value.Elem(), parameters)
	case *types.Chan:
		result := storageUnaryType(storageTypeChannel, value.Elem(), parameters)
		result.Direction = int(value.Dir())
		return result
	case *types.Struct:
		result := StorageEffectType{Kind: storageTypeStruct}
		for index := 0; index < value.NumFields(); index++ {
			field := value.Field(index)
			result.Fields = append(result.Fields, encodeStorageEffectType(field.Type(), parameters))
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
	case *types.Signature:
		return StorageEffectType{Kind: storageTypeSignature, Variadic: value.Variadic(),
			Parameters: encodeStorageTuple(value.Params(), parameters),
			Results:    encodeStorageTuple(value.Results(), parameters)}
	case *types.Named:
		result := StorageEffectType{Kind: storageTypeNamed, Name: value.Obj().Name()}
		if value.Obj().Pkg() != nil {
			result.Package = value.Obj().Pkg().Path()
		}
		for index := 0; index < value.TypeArgs().Len(); index++ {
			result.Arguments = append(result.Arguments,
				encodeStorageEffectType(value.TypeArgs().At(index), parameters))
		}
		return result
	case *types.Interface:
		value = value.Complete()
		result := StorageEffectType{Kind: storageTypeInterface}
		for index := 0; index < value.NumExplicitMethods(); index++ {
			method := value.ExplicitMethod(index)
			result.Fields = append(result.Fields,
				encodeStorageEffectType(method.Type(), parameters))
			result.FieldNames = append(result.FieldNames, method.Name())
			packagePath := ""
			if method.Pkg() != nil {
				packagePath = method.Pkg().Path()
			}
			result.FieldPkgs = append(result.FieldPkgs, packagePath)
		}
		for index := 0; index < value.NumEmbeddeds(); index++ {
			result.Element = append(result.Element,
				encodeStorageEffectType(value.EmbeddedType(index), parameters))
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
) StorageEffectType {
	return StorageEffectType{Kind: kind,
		Element: []StorageEffectType{encodeStorageEffectType(element, parameters)}}
}

func encodeStorageTuple(
	tuple *types.Tuple,
	parameters map[*types.TypeParam]zeroParameter,
) []StorageEffectType {
	result := make([]StorageEffectType, tuple.Len())
	for index := range result {
		result[index] = encodeStorageEffectType(tuple.At(index).Type(), parameters)
	}
	return result
}

func decodeStorageEffectType(
	projection StorageEffectType,
	receiverArguments []types.Type,
	typeArguments []types.Type,
) types.Type {
	switch projection.Kind {
	case storageTypeParameter:
		return storageBoundType(typeArguments, projection.Parameter)
	case storageTypeReceiver:
		return storageBoundType(receiverArguments, projection.Parameter)
	case storageTypeBasic:
		return types.Typ[types.BasicKind(projection.Basic)]
	case storageTypeSlice:
		return types.NewSlice(decodeStorageElement(projection, receiverArguments, typeArguments))
	case storageTypeArray:
		return types.NewArray(decodeStorageElement(projection, receiverArguments, typeArguments), projection.Length)
	case storageTypeMap:
		return types.NewMap(
			decodeStorageTypeList(projection.Key, receiverArguments, typeArguments)[0],
			decodeStorageElement(projection, receiverArguments, typeArguments),
		)
	case storageTypePointer:
		return types.NewPointer(decodeStorageElement(projection, receiverArguments, typeArguments))
	case storageTypeChannel:
		return types.NewChan(types.ChanDir(projection.Direction),
			decodeStorageElement(projection, receiverArguments, typeArguments))
	case storageTypeStruct:
		fields := make([]*types.Var, len(projection.Fields))
		for index, field := range decodeStorageTypeList(projection.Fields, receiverArguments, typeArguments) {
			var pkg *types.Package
			if projection.FieldPkgs[index] != "" {
				pkg = types.NewPackage(projection.FieldPkgs[index], projection.FieldPkgs[index])
			}
			fields[index] = types.NewField(0, pkg, projection.FieldNames[index], field, projection.FieldEmbed[index])
		}
		return types.NewStruct(fields, projection.FieldTags)
	case storageTypeSignature:
		parameters := types.NewTuple(storageTypeVars(
			projection.Parameters, receiverArguments, typeArguments)...)
		results := types.NewTuple(storageTypeVars(
			projection.Results, receiverArguments, typeArguments)...)
		return types.NewSignatureType(nil, nil, nil, parameters, results, projection.Variadic)
	case storageTypeNamed:
		return decodeStorageNamed(projection, receiverArguments, typeArguments)
	case storageTypeInterface:
		methods := make([]*types.Func, len(projection.Fields))
		for index, signature := range decodeStorageTypeList(
			projection.Fields, receiverArguments, typeArguments,
		) {
			var pkg *types.Package
			if projection.FieldPkgs[index] != "" {
				pkg = types.NewPackage(projection.FieldPkgs[index], projection.FieldPkgs[index])
			}
			methods[index] = types.NewFunc(0, pkg, projection.FieldNames[index], signature.(*types.Signature))
		}
		embedded := decodeStorageTypeList(projection.Element, receiverArguments, typeArguments)
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
) types.Type {
	return decodeStorageTypeList(projection.Element, receiverArguments, typeArguments)[0]
}

func decodeStorageTypeList(
	projections []StorageEffectType,
	receiverArguments []types.Type,
	typeArguments []types.Type,
) []types.Type {
	result := make([]types.Type, len(projections))
	for index, projection := range projections {
		result[index] = decodeStorageEffectType(projection, receiverArguments, typeArguments)
	}
	return result
}

func storageTypeVars(
	projections []StorageEffectType,
	receiverArguments []types.Type,
	typeArguments []types.Type,
) []*types.Var {
	typesList := decodeStorageTypeList(projections, receiverArguments, typeArguments)
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
) types.Type {
	var pkg *types.Package
	if projection.Package != "" {
		pkg = types.NewPackage(projection.Package, projection.Package)
	}
	named := types.NewNamed(types.NewTypeName(0, pkg, projection.Name, nil),
		types.NewStruct(nil, nil), nil)
	arguments := decodeStorageTypeList(projection.Arguments, receiverArguments, typeArguments)
	if len(arguments) == 0 {
		return named
	}
	parameters := make([]*types.TypeParam, len(arguments))
	for index := range parameters {
		parameters[index] = types.NewTypeParam(
			types.NewTypeName(0, nil, "T", nil),
			types.NewInterfaceType(nil, nil).Complete(),
		)
	}
	named.SetTypeParams(parameters)
	result, err := types.Instantiate(nil, named, arguments, false)
	if err != nil {
		return named
	}
	return result
}
