#!/usr/bin/env python3
"""Compare GTM UI export vs MCP export to find structural differences."""
import json

MCP_FILE = "/home/raul/Descargas/GTM-T2TC8MV_v1274.json"
UI_FILE = "/home/raul/Descargas/GTM-T2TC8MV_v1274 (1).json"

def load_json(path):
    with open(path, 'r') as f:
        content = f.read()
    try:
        data = json.loads(content)
    except json.JSONDecodeError:
        idx = content.find('{')
        if idx >= 0:
            data = json.loads(content[idx:])
        else:
            raise
    if isinstance(data, list):
        for item in data:
            if isinstance(item, dict) and 'text' in item:
                try:
                    inner = json.loads(item['text'])
                    if isinstance(inner, dict) and 'containerVersion' in inner:
                        return inner
                except:
                    pass
        raise ValueError("Could not find containerVersion in content blocks")
    return data

print("Loading MCP export...")
mcp = load_json(MCP_FILE)
print(f"  Type: {type(mcp).__name__}, keys: {list(mcp.keys())}")

print("Loading UI export...")
ui = load_json(UI_FILE)
print(f"  Type: {type(ui).__name__}, keys: {list(ui.keys())}")

print("\n=== TOP-LEVEL KEY COMPARISON ===")
mcp_keys = set(mcp.keys())
ui_keys = set(ui.keys())
print(f"MCP only: {mcp_keys - ui_keys}")
print(f"UI only: {ui_keys - mcp_keys}")

if 'containerVersion' in mcp and 'containerVersion' in ui:
    mcv = mcp['containerVersion']
    ucv = ui['containerVersion']
    print(f"\n=== containerVersion KEYS ===")
    mcv_keys = set(mcv.keys())
    ucv_keys = set(ucv.keys())
    print(f"MCP only: {mcv_keys - ucv_keys}")
    print(f"UI only: {ucv_keys - mcv_keys}")
    
    for key in sorted(mcv_keys | ucv_keys):
        m_val = mcv.get(key)
        u_val = ucv.get(key)
        if isinstance(m_val, list) and isinstance(u_val, list):
            print(f"\n  {key}: MCP={len(m_val)} items, UI={len(u_val)} items")
            if m_val and u_val:
                m_item_keys = set(m_val[0].keys()) if isinstance(m_val[0], dict) else set()
                u_item_keys = set(u_val[0].keys()) if isinstance(u_val[0], dict) else set()
                extra = m_item_keys - u_item_keys
                missing = u_item_keys - m_item_keys
                if extra:
                    print(f"    MCP EXTRA fields: {extra}")
                if missing:
                    print(f"    MCP MISSING fields: {missing}")

    # Check ALL entities for extra fields
    print("\n=== EXTRA FIELDS PER ENTITY TYPE (sample from all items) ===")
    for entity_type in ['tag', 'trigger', 'variable', 'customTemplate', 'builtInVariable', 'folder']:
        m_entities = mcv.get(entity_type, [])
        u_entities = ucv.get(entity_type, [])
        if not m_entities or not u_entities:
            continue
        # Gather all keys across ALL items
        m_all_keys = set()
        u_all_keys = set()
        for e in m_entities:
            if isinstance(e, dict):
                m_all_keys |= set(e.keys())
        for e in u_entities:
            if isinstance(e, dict):
                u_all_keys |= set(e.keys())
        extra = m_all_keys - u_all_keys
        missing = u_all_keys - m_all_keys
        if extra or missing:
            print(f"\n  {entity_type}:")
            if extra: print(f"    MCP EXTRA: {extra}")
            if missing: print(f"    MCP MISSING: {missing}")

# Check for hidden parameters
print("\n=== HIDDEN PARAMETER CHECK ===")
if 'containerVersion' in mcp:
    cv = mcp['containerVersion']
    for entity_type in ['tag', 'trigger', 'variable', 'customTemplate']:
        entities = cv.get(entity_type, [])
        hidden_count = 0
        for e in entities:
            if isinstance(e, dict):
                for p in e.get('parameter', []):
                    if isinstance(p, dict) and p.get('type') == 'HIDDEN':
                        hidden_count += 1
                        if hidden_count <= 3:
                            print(f"  HIDDEN param in {entity_type} '{e.get('name','?')}': key={p.get('key')}")
        if hidden_count > 3:
            print(f"  ... {hidden_count} total HIDDEN params in {entity_type}")

# Check customTemplate specifically
print("\n=== CUSTOM TEMPLATE COMPARISON ===")
m_templates = mcv.get('customTemplate', [])
u_templates = ucv.get('customTemplate', [])
print(f"MCP: {len(m_templates)} templates, UI: {len(u_templates)} templates")
if m_templates:
    # Sample first template
    mt = m_templates[0]
    for ut in u_templates:
        if ut.get('name') == mt.get('name'):
            m_keys = set(mt.keys())
            u_keys = set(ut.keys())
            print(f"  Template '{mt.get('name')}':")
            print(f"    MCP keys: {sorted(m_keys)}")
            print(f"    UI keys: {sorted(u_keys)}")
            print(f"    MCP EXTRA: {m_keys - u_keys}")
            print(f"    MCP MISSING: {u_keys - m_keys}")
            break
