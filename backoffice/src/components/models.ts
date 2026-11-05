export interface Todo {
  id: number;
  content: string;
}

export interface Meta {
  totalCount: number;
}

export interface Device {
  id: number;
  type: string;
  url: string;
  name: string;
  config: string;
  periodicity: string | null;
  enabled: boolean;
}

export interface Tag {
  id: number;
  name: string;
  type: string;
  valueType: string;
  valueStorage: string; // insert, update
  config: string;
  deviceId: number;
  deviceName: string | undefined;
  devicePeriodicity: string | undefined;
  periodicityCount: number;
  expiresIn: number; // hours
  displayIn: string[];
  enabled: boolean;
}

export interface TagValueFloat {
  tagId: number;
  time: string;
  value: number;
}

// Automations: the same JSON as the backend (models.Automation in backend-go)
export type AutomationNodeType =
  | 'timer'
  | 'sunrise'
  | 'sunset'
  | 'tagValueFloat'
  | 'tagValueBoolean'
  | 'and'
  | 'or'
  | 'condition'
  | 'waitCheck'
  | 'whatsapp'
  | 'device'
  | 'http';

export interface AutomationNode {
  id: string;
  type: AutomationNodeType;
  cron?: string;
  offsetMinutes?: number; // sunrise/sunset: minutes after (negative: before)
  tagId?: number;
  operator?: '>' | '<' | '=' | '>=' | '<=';
  value?: number;
  boolValue?: boolean; // boolean tags: the value that matches
  delaySeconds?: number; // waitCheck
  text?: string;
  deviceId?: number;
  on?: boolean;
  url?: string;
  method?: string;
  body?: string;
  headers?: Record<string, string>;
  ui?: { position?: { x: number; y: number } };
}

export interface AutomationEdge {
  from: string;
  to: string;
}

export interface Automation {
  id: number;
  name: string;
  enabled: boolean;
  graph: { nodes: AutomationNode[]; edges: AutomationEdge[] };
}
