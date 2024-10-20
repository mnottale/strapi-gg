#! /usr/bin/env python3

import sys
import requests
import json

strapi_api = sys.argv[1]
with open(sys.argv[2], 'r') as fd:
    strapi_token = fd.read()
if strapi_token[-1] == '\n':
    strapi_token = strapi_token[:-1]

resp = requests.get(strapi_api + '/content-type-builder/content-types',
    headers = {'Authorization': 'Bearer ' + strapi_token, 'Accept': 'application/json'})
cts = resp.json()

def to_pascal_case(s):
    next_upper = True
    res = ''
    for c in s:
        if next_upper:
            res += c.upper()
            next_upper = False
        elif c == '_' or c == '-':
            next_upper = True
        else:
            res += c
    return res

def generate_all(cts, out):
    uid_map = dict()
    IND='    '
    type_map = {
        'integer': 'int',
        'string': 'string',
        'text': 'string',
        'datetime': 'time.Time',
        'date': 'time.Time',
        'boolean': 'bool',
        'richtext': 'string',
    }
    for ct in cts['data']:
        api = ct['apiID']
        uid = ct['uid']
        uid_map[uid] = api

    print('package main\nimport "time"\n\n', file=out)
    found = set()
    for ct in cts['data']:
        api = ct['apiID']
        collection_name = ct['schema']['pluralName']
        cnn = to_pascal_case(api)
        cn = cnn
        cnt = 1
        while cn in found:
            cn = cnn + str(cnt)
            cnt += 1
        found.add(cn)
        omain = """
type {cn}Response struct {{
    Meta Meta `json:"meta"`
    Data []{cn} `json:"data"`
}}

func (r *{cn}Response) NewInstance() StrapiResponse {{
    return &{cn}Response{{}}
}}

func (r *{cn}Response) PageCount() int {{
    return r.Meta.Pagination.PageCount
}}

func (r *{cn}Response) Add(next StrapiResponse) {{
    r.Data = append(r.Data, next.(*{cn}Response).Data...)
}}

func (r *{cn}Response) ResourceName() string {{
    return "{rn}"
}}

type {cn}Ptr struct {{
   Data *{cn} `json:"data"`
}}

func (r *{cn}Ptr) PtrResourceName() string {{
    return "{rn}"
}}

type {cn}ArrayPtr struct {{
   Data []*{cn} `json:"data"`
}}

type {cn} struct {{
    Id    int            `json:"id"`
    Attrs *{cn}Attrs     `json:"attributes"`
}}

func (r *{cn}) ResourceName() string {{
    return "{rn}"
}}

func (r *{cn}WriteAttrs) WriterResourceName() string {{
    return "{rn}"
}}

func (r *{cn}NullableWriteAttrs) WriterResourceName() string {{
    return "{rn}"
}}

func (r *{cn}) GetId() int {{
    return r.Id
}}

""".format(**{'cn':cn, 'rn': collection_name})
        oread = 'type {}Attrs struct {{\n'.format(cn)
        owrite = 'type {}WriteAttrs struct {{\n'.format(cn)
        onulwrite = 'type {}NullableWriteAttrs struct {{\n'.format(cn)
        ocp = 'func (r *{}) AsWriter() interface{{}} {{ res := &{}WriteAttrs{{}}\n'.format(cn, cn)
        onulcp = 'func (r *{}) AsNullableWriter() interface{{}} {{ res := &{}NullableWriteAttrs{{}}\n'.format(cn, cn)
        for k, v in ct['schema']['attributes'].items():
            fn = to_pascal_case(k)
            ct = v['type']
            rel = False
            mult = v.get('multiple', False)
            req = v.get('required', False)
            if ct in type_map:
                ft = type_map[ct]
            elif ct == 'relation':
                if 'target' not in v:
                    print('No target for relation {}.{}: {}'.format(cn, k, json.dumps(v)), file=sys.stderr)
                    continue
                if v['target'] not in uid_map:
                    print('Type not found: {}'.format(v['target']))
                    continue
                rel = True
                rel_mult = v['relation'] == 'manyToMany'
                ft = '*' + to_pascal_case(uid_map[v['target']]) + (rel_mult and 'ArrayPtr' or 'Ptr')
            else:
                print('Dropping {}.{} {}'.format(cn, k, ct), file=sys.stderr)
                continue
            if mult:
                ft = '[]'+ft
            ftw = ft
            ftwn = ft
            jextra=''
            if ct == 'relation':
                ftw = 'int'
                ftwn = '*int'
                jextra = ',omitempty'
                rel_mult = v['relation'] == 'manyToMany'
                if rel_mult:
                    ftw = '[]int'
                    ftwn = '[]int'
                if not rel_mult:
                    ocp +=    IND*2 + 'if r.Attrs.{} != nil && r.Attrs.{}.Data != nil {{\n        res.{} = r.Attrs.{}.Data.Id\n     }}\n'.format(fn, fn, fn, fn)
                    onulcp += IND*2 + 'if r.Attrs.{} != nil && r.Attrs.{}.Data != nil {{\n        res.{} = IntPtr(r.Attrs.{}.Data.Id)\n     }}\n'.format(fn, fn, fn, fn)
                else:
                    add = IND*2 + 'if r.Attrs.{fn} != nil && r.Attrs.{fn}.Data != nil {{\n            for _, v := range r.Attrs.{fn}.Data {{\n           res.{fn} = append(res.{fn},  v.Id)\n }}\n    }}\n'.format(fn=fn)
                    ocp += add
                    onulcp += add
            else:
                ocp += IND*2 + 'res.{} =  r.Attrs.{}\n'.format(fn, fn)
                onulcp += IND*2 + 'res.{} =  r.Attrs.{}\n'.format(fn, fn)
            oread += IND + '{} {} `json:"{}"`\n'.format(fn, ft, k)
            owrite += IND + '{} {} `json:"{}{}"`\n'.format(fn, ftw, k, jextra)
            onulwrite += IND + '{} {} `json:"{}"`\n'.format(fn, ftwn, k)
        oread += '}\n'
        owrite += '}\n'
        onulwrite += '}\n'
        ocp += '    return res\n}\n'
        onulcp += '    return res\n}\n'
        print(omain, file=out)
        print(oread, file=out)
        print(owrite, file=out)
        print(onulwrite, file=out)
        print(ocp, file=out)
        print(onulcp, file=out)
generate_all(cts, sys.stdout)
#print(resp.status_code)
#print(json.dumps(resp.json()))