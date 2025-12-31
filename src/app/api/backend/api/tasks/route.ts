import { NextResponse } from 'next/server';

// Mock database in memory
let tasks = [
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

export async function GET(request: Request) {
  const { searchParams } = new URL(request.url);
  const status = searchParams.get('status');
  
  let filteredTasks = tasks;
  if (status) {
    filteredTasks = tasks.filter(t => t.status === status);
  }
  
  return NextResponse.json({
    data: filteredTasks,
    total: filteredTasks.length,
  });
}

export async function POST(request: Request) {
  try {
    const body = await request.json();
    
    if (!body.title) {
      return NextResponse.json({ message: 'Title is required' }, { status: 400 });
    }
    
    const newTask = {
      id: Math.random().toString(36).substring(2, 9),
      title: body.title,
      description: body.description || '',
      status: body.status || 'todo',
      priority: body.priority || 'medium',
      createdAt: new Date().toISOString(),
    };
    
    tasks = [newTask, ...tasks];
    
    return NextResponse.json(newTask, { status: 201 });
  } catch (error) {
    return NextResponse.json({ message: 'Invalid payload' }, { status: 400 });
  }
}
