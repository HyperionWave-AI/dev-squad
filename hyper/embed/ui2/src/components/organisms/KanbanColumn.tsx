import { Droppable } from '@hello-pangea/dnd';
import { Card, CardHeader, CardTitle } from '@/components/molecules/Card';
import { Badge } from '@/components/atoms/Badge';
import { TaskCard } from './TaskCard';
import { cn } from '@/utils';
import type { FlattenedTask, TaskStatus } from '@/types/coordinator';

interface KanbanColumnProps {
  id: TaskStatus;
  title: string;
  tasks: FlattenedTask[];
  color: string;
  bgColor: string;
  onTaskClick: (task: FlattenedTask) => void;
}

export function KanbanColumn({ id, title, tasks, color, bgColor, onTaskClick }: KanbanColumnProps) {
  return (
    <div className="flex flex-col min-w-[300px] flex-1">
      {/* Column Header */}
      <Card className="mb-3" style={{ backgroundColor: bgColor, borderTop: `4px solid ${color}` }}>
        <CardHeader className="p-4">
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg font-semibold" style={{ color }}>
              {title}
            </CardTitle>
            <Badge
              variant="secondary"
              className="text-xs font-semibold"
              style={{ backgroundColor: color, color: 'white' }}
            >
              {tasks.length}
            </Badge>
          </div>
        </CardHeader>
      </Card>

      {/* Droppable Area */}
      <Droppable droppableId={id}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.droppableProps}
            className={cn(
              'flex-1 min-h-[200px] rounded-lg p-2 transition-colors',
              snapshot.isDraggingOver && 'bg-gray-100 dark:bg-gray-800'
            )}
          >
            {tasks.map((task, index) => (
              <TaskCard key={task.id} task={task} index={index} onClick={onTaskClick} />
            ))}
            {provided.placeholder}
          </div>
        )}
      </Droppable>
    </div>
  );
}
