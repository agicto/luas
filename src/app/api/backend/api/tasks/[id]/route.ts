import { NextResponse } from 'next/server';
import { readTasks, writeTasks } from '../db';

export async function GET(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;
  const tasks = await readTasks();
  const task = tasks.find(t => t.id === id);
  
  if (!task) {
    return NextResponse.json({ message: 'Task not found' }, { status: 404 });
  }
  
  return NextResponse.json(task);
}

export async function PATCH(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;
  const body = await request.json();
  const tasks = await readTasks();
  const index = tasks.findIndex(t => t.id === id);
  
  if (index === -1) {
    return NextResponse.json({ message: 'Task not found' }, { status: 404 });
  }
  
  const updatedTask = {
    ...tasks[index],
    ...body,
    // Ensure ID doesn't change
    id, 
  };
  
  const updatedTasks = [...tasks];
  updatedTasks[index] = updatedTask;
  await writeTasks(updatedTasks);
  
  return NextResponse.json(updatedTask);
}

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  const { id } = await params;
  const tasks = await readTasks();
  const index = tasks.findIndex(t => t.id === id);
  
  if (index === -1) {
    return NextResponse.json({ message: 'Task not found' }, { status: 404 });
  }
  
  const updatedTasks = tasks.filter(t => t.id !== id);
  await writeTasks(updatedTasks);
  
  return new NextResponse(null, { status: 204 });
}
