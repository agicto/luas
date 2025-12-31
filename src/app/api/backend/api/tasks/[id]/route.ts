import { NextResponse } from 'next/server';

// In a real mock, this would be shared with the /tasks route.
// For simplicity in this demo, we'll re-define it, but in a production scaffold 
// you'd use a shared memory store or a local JSON file.
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

export async function GET(
  request: Request,
  { params }: { params: { id: string } }
) {
  const task = tasks.find(t => t.id === params.id);
  
  if (!task) {
    return NextResponse.json({ message: 'Task not found' }, { status: 404 });
  }
  
  return NextResponse.json(task);
}

export async function PATCH(
  request: Request,
  { params }: { params: { id: string } }
) {
  const body = await request.json();
  const index = tasks.findIndex(t => t.id === params.id);
  
  if (index === -1) {
    return NextResponse.json({ message: 'Task not found' }, { status: 404 });
  }
  
  const updatedTask = {
    ...tasks[index],
    ...body,
    // Ensure ID doesn't change
    id: params.id, 
  };
  
  tasks[index] = updatedTask;
  
  return NextResponse.json(updatedTask);
}

export async function DELETE(
  request: Request,
  { params }: { params: { id: string } }
) {
  const index = tasks.findIndex(t => t.id === params.id);
  
  if (index === -1) {
    return NextResponse.json({ message: 'Task not found' }, { status: 404 });
  }
  
  tasks = tasks.filter(t => t.id !== params.id);
  
  return new NextResponse(null, { status: 204 });
}
