import fs from 'fs/promises';
import path from 'path';

// Define the path to the JSON file
const DB_PATH = path.join(process.cwd(), 'src/app/api/backend/api/tasks/tasks.json');

export interface Task {
  id: string;
  title: string;
  description?: string;
  status: 'todo' | 'doing' | 'done';
  priority: 'low' | 'medium' | 'high';
  createdAt: string;
}

// Initial default data
const DEFAULT_TASKS: Task[] = [
  {
    id: '1',
    title: 'Design System Polish',
    description: 'Update the card components with new surface tokens.',
    status: 'doing',
    priority: 'high',
    createdAt: new Date().toISOString(),
  },
  {
    id: '2',
    title: 'Internationalization',
    description: 'Translate the settings page to Japanese.',
    status: 'todo',
    priority: 'medium',
    createdAt: new Date().toISOString(),
  },
];

/**
 * Read tasks from the JSON file
 */
export async function readTasks(): Promise<Task[]> {
  try {
    const data = await fs.readFile(DB_PATH, 'utf-8');
    return JSON.parse(data);
  } catch (error) {
    // If file doesn't exist, create it with default data
    await writeTasks(DEFAULT_TASKS);
    return DEFAULT_TASKS;
  }
}

/**
 * Write tasks to the JSON file
 */
export async function writeTasks(tasks: Task[]): Promise<void> {
  try {
    // Ensure the directory exists
    const dir = path.dirname(DB_PATH);
    await fs.mkdir(dir, { recursive: true });
    
    await fs.writeFile(DB_PATH, JSON.stringify(tasks, null, 2), 'utf-8');
  } catch (error) {
    console.error('Failed to write tasks to DB:', error);
  }
}
