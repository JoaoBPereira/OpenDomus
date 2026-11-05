import type { Edge, Node } from '@vue-flow/core';
import cronstrue from 'cronstrue';
import type { Automation, AutomationNode, AutomationNodeType } from '@/components/models';

export type NodeCategory = 'trigger' | 'gate' | 'action';

export const nodeTypes: {
  type: AutomationNodeType;
  label: string;
  icon: string;
  category: NodeCategory;
}[] = [
  { type: 'timer', label: 'Timer', icon: 'schedule', category: 'trigger' },
  { type: 'sunrise', label: 'Sunrise', icon: 'wb_sunny', category: 'trigger' },
  { type: 'sunset', label: 'Sunset', icon: 'wb_twilight', category: 'trigger' },
  { type: 'tagValueFloat', label: 'Tag value', icon: 'sensors', category: 'trigger' },
  { type: 'tagValueBoolean', label: 'Tag on/off', icon: 'toggle_on', category: 'trigger' },
  { type: 'and', label: 'AND', icon: 'join_inner', category: 'gate' },
  { type: 'or', label: 'OR', icon: 'join_full', category: 'gate' },
  { type: 'condition', label: 'Condition', icon: 'rule', category: 'gate' },
  { type: 'waitCheck', label: 'Wait & check', icon: 'hourglass_bottom', category: 'gate' },
  { type: 'whatsapp', label: 'WhatsApp', icon: 'chat', category: 'action' },
  { type: 'device', label: 'Device', icon: 'power_settings_new', category: 'action' },
  { type: 'http', label: 'HTTP request', icon: 'http', category: 'action' },
];

export function nodeInfo(type: AutomationNodeType) {
  return nodeTypes.find((info) => info.type === type) ?? nodeTypes[0]!;
}

// The Vue Flow node type is the category: one FlowNode component draws all of them
export function toFlow(automation: Automation): { nodes: Node<AutomationNode>[]; edges: Edge[] } {
  return {
    nodes: automation.graph.nodes.map((node, index) => ({
      id: node.id,
      type: nodeInfo(node.type).category,
      position: node.ui?.position ?? { x: 50 + index * 40, y: 50 + index * 40 },
      data: { ...node },
    })),
    edges: automation.graph.edges.map((edge) => ({
      id: `${edge.from}->${edge.to}`,
      source: edge.from,
      target: edge.to,
    })),
  };
}

export function fromFlow(nodes: Node<AutomationNode>[], edges: Edge[]): Automation['graph'] {
  return {
    // every node of the editor is created with its data
    nodes: nodes.map((node) => ({
      ...(node.data as AutomationNode),
      id: node.id,
      ui: {
        ...node.data?.ui,
        position: { x: Math.round(node.position.x), y: Math.round(node.position.y) },
      },
    })),
    edges: edges.map((edge) => ({ from: edge.source, to: edge.target })),
  };
}

export function newNode(type: AutomationNodeType): AutomationNode {
  const id = `${type}-${Math.random().toString(36).slice(2, 8)}`;
  switch (type) {
    case 'timer':
      return { id, type, cron: '0 8 * * *' };
    case 'sunrise':
    case 'sunset':
      return { id, type, offsetMinutes: 0 };
    case 'tagValueFloat':
    case 'condition':
      return { id, type, tagId: 0, operator: '>', value: 0, boolValue: true };
    case 'tagValueBoolean':
      return { id, type, tagId: 0, boolValue: true };
    case 'waitCheck':
      return { id, type, tagId: 0, operator: '>', value: 0, boolValue: true, delaySeconds: 60 };
    case 'whatsapp':
      return { id, type, text: '' };
    case 'device':
      return { id, type, deviceId: 0, on: true };
    case 'http':
      return { id, type, method: 'GET', url: '', body: '', headers: {} };
    default:
      return { id, type };
  }
}

// Same rule as the backend: triggers have no input, actions have no output
export function canConnect(sourceType: AutomationNodeType, targetType: AutomationNodeType) {
  return nodeInfo(sourceType).category !== 'action' && nodeInfo(targetType).category !== 'trigger';
}

export function formatCron(cron?: string) {
  if (!cron) return 'No cron';
  try {
    return cronstrue.toString(cron, {
      throwExceptionOnParseError: true,
      use24HourTimeFormat: true,
    });
  } catch {
    return 'Invalid cron';
  }
}

// "At sunset", "30 min before sunrise"
export function formatOffset(type: 'sunrise' | 'sunset', offsetMinutes = 0) {
  if (!offsetMinutes) return `At ${type}`;
  return `${Math.abs(offsetMinutes)} min ${offsetMinutes < 0 ? 'before' : 'after'} ${type}`;
}

// The triggers listed in "Automations Time" and "Automations Values"
export const timeTriggers: AutomationNodeType[] = ['timer', 'sunrise', 'sunset'];
export const valueTriggers: AutomationNodeType[] = ['tagValueFloat', 'tagValueBoolean'];

type Names = {
  tags?: Map<number, string>;
  tagTypes?: Map<number, string>;
  devices?: Map<number, string>;
};

// "Boiler <= 45" for a float tag, "Gate is on" for a boolean tag
function tagCheck(node: AutomationNode, names: Names) {
  const tag = names.tags?.get(node.tagId ?? 0) ?? `Tag ${node.tagId}`;
  if (
    node.type === 'tagValueBoolean' ||
    names.tagTypes?.get(node.tagId ?? 0) === 'tagvalueboolean'
  ) {
    return `${tag} is ${node.boolValue ? 'on' : 'off'}`;
  }
  return `${tag} ${node.operator} ${node.value}`;
}

// "90 s", "5 min", "1 h 30 min"
export function formatSeconds(seconds = 0) {
  if (seconds < 60) return `${seconds} s`;
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const rest = seconds % 60;
  return [hours && `${hours} h`, minutes && `${minutes} min`, rest && `${rest} s`]
    .filter(Boolean)
    .join(' ');
}

// One readable line per node, for the canvas and the lists
export function summary(node: AutomationNode, names: Names = {}) {
  switch (node.type) {
    case 'timer':
      return formatCron(node.cron);
    case 'sunrise':
    case 'sunset':
      return formatOffset(node.type, node.offsetMinutes);
    case 'tagValueFloat':
    case 'tagValueBoolean':
      return tagCheck(node, names);
    case 'condition':
      return `If ${tagCheck(node, names)}`;
    case 'waitCheck':
      return `After ${formatSeconds(node.delaySeconds)}, if ${tagCheck(node, names)}`;
    case 'whatsapp':
      return node.text || 'No text';
    case 'device':
      return `${names.devices?.get(node.deviceId ?? 0) ?? `Device ${node.deviceId}`} ${node.on ? 'on' : 'off'}`;
    case 'http':
      return `${node.method || 'GET'} ${node.url || 'no url'}`;
    default:
      return '';
  }
}
